package crudp

import (
	"context"
	"reflect"

	. "github.com/cdvelop/tinystring"
)

// Packet represents both requests and responses of the protocol
type Packet struct {
	Action    byte     `json:"action"`
	HandlerID uint8    `json:"handler_id"`
	ReqID     string   `json:"req_id"`
	Data      [][]byte `json:"data"`
}

// BatchRequest is what is sent in the POST /sync
type BatchRequest struct {
	Packets []Packet `json:"packets"`
}

// BatchResponse is what is received by SSE
type BatchResponse struct {
	Results []PacketResult `json:"results"`
}

type PacketResult struct {
	Packet             // Embed Packet complete for symmetry with BatchRequest
	MessageType uint8  `json:"message_type"` // tinystring.MessageType (0=Normal, 1=Info, 2=Error, 3=Warning, 4=Success)
	Message     string `json:"message"`      // Message for the user
}

// EncodePacket encodes a packet for a known handler using this CrudP's codec instance
func (cp *CrudP) EncodePacket(action byte, handlerID uint8, reqID string, data ...any) ([]byte, error) {
	encoded := make([][]byte, 0, len(data))
	for _, item := range data {
		bytes, err := cp.codec.Encode(item)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, bytes)
	}

	packet := Packet{
		Action:    action,
		HandlerID: handlerID,
		ReqID:     reqID,
		Data:      encoded,
	}

	return cp.codec.Encode(packet)
}

// DecodePacket decodes a packet using this CrudP's codec instance
func (cp *CrudP) DecodePacket(data []byte, packet *Packet) error {
	return cp.codec.Decode(data, packet)
}

// DecodeData decodes the packet data using this CrudP's codec instance
func (cp *CrudP) DecodeData(packet *Packet, index int, target any) error {
	if index >= len(packet.Data) {
		return Errf("index out of range")
	}
	return cp.codec.Decode(packet.Data[index], target)
}

// ProcessBatch automatically processes a batch of packets and returns batch results
func (cp *CrudP) ProcessBatch(ctx context.Context, requestBytes []byte) ([]byte, error) {
	cp.log("ProcessBatch called with bytes:", len(requestBytes))
	var batchReq BatchRequest
	if err := cp.codec.Decode(requestBytes, &batchReq); err != nil {
		cp.log("ProcessBatch decode error:", err)
		return cp.createErrorBatchResponse("decode_error", err)
	}

	cp.log("ProcessBatch decoded packets:", len(batchReq.Packets))

	results := make([]PacketResult, 0, len(batchReq.Packets))

	for _, packet := range batchReq.Packets {
		result, err := cp.processSinglePacket(ctx, &packet)
		results = append(results, result)
		if err != nil {
			// Continue processing other packets even if one fails
			continue
		}
	}

	batchResp := BatchResponse{
		Results: results,
	}

	return cp.codec.Encode(batchResp)
}

func (cp *CrudP) processSinglePacket(ctx context.Context, packet *Packet) (PacketResult, error) {
	pr := PacketResult{
		Packet: *packet, // Embed original packet (includes Data [][]byte)
	}

	// Decode data with known types
	decodedData, err := cp.decodeWithKnownType(packet, packet.HandlerID)
	if err != nil {
		pr.MessageType = uint8(Msg.Error)
		pr.Message = err.Error()
		return pr, err
	}

	// Call handler
	result, err := cp.CallHandler(ctx, packet.HandlerID, packet.Action, decodedData...)
	if err != nil {
		cp.log("processSinglePacket CallHandler error:", err)
		pr.MessageType = uint8(Msg.Error)
		pr.Message = err.Error()
		return pr, err
	}

	cp.log("processSinglePacket CallHandler success, result type:", reflect.TypeOf(result))

	// Process result - can be multiple Response
	if err := cp.encodeResultToPacket(&pr, result); err != nil {
		pr.MessageType = uint8(Msg.Error)
		pr.Message = err.Error()
		return pr, err
	}

	pr.MessageType = uint8(Msg.Success)
	pr.Message = "OK"
	return pr, nil
}

// encodeResultToPacket encodes handler result to Data [][]byte
func (cp *CrudP) encodeResultToPacket(pr *PacketResult, result any) error {
	if result == nil {
		return nil
	}

	// Case 1: Slice of Response for multiple broadcast
	cp.log("encodeResultToPacket result type:", reflect.TypeOf(result).String())
	if responses, ok := result.([]Response); ok {
		pr.Data = make([][]byte, 0, len(responses))
		for _, resp := range responses {
			data, broadcast, err := resp.Response()
			if err != nil {
				return err
			}

			// SSE routing if broadcast targets exist
			if len(broadcast) > 0 {
				cp.routeToSSE(data, broadcast, pr.HandlerID)
			}

			encoded, err := cp.codec.Encode(data)
			if err != nil {
				return err
			}
			pr.Data = append(pr.Data, encoded)
		}
		return nil
	}

	// Case 2: Individual Response
	if resp, ok := result.(Response); ok {
		data, broadcast, err := resp.Response()
		if err != nil {
			return err
		}

		if len(broadcast) > 0 {
			cp.routeToSSE(data, broadcast, pr.HandlerID)
		}

		encoded, err := cp.codec.Encode(data)
		if err != nil {
			return err
		}
		pr.Data = [][]byte{encoded}
		return nil
	}

	// Case 3: Direct value
	encoded, err := cp.codec.Encode(result)
	if err != nil {
		return err
	}
	pr.Data = [][]byte{encoded}
	return nil
}

func (cp *CrudP) createErrorBatchResponse(reqID string, err error) ([]byte, error) {
	result := PacketResult{
		Packet:      Packet{ReqID: reqID},
		MessageType: uint8(Msg.Error),
		Message:     err.Error(),
	}

	return cp.codec.Encode(BatchResponse{Results: []PacketResult{result}})
}

// ProcessPacket processes a single packet (for backward compatibility)
// Internally wraps in a batch and processes
func (cp *CrudP) ProcessPacket(ctx context.Context, requestBytes []byte) ([]byte, error) {
	var packet Packet
	if err := cp.DecodePacket(requestBytes, &packet); err != nil {
		return nil, err
	}

	batchReq := BatchRequest{Packets: []Packet{packet}}
	batchBytes, err := cp.codec.Encode(batchReq)
	if err != nil {
		return nil, err
	}

	batchRespBytes, err := cp.ProcessBatch(ctx, batchBytes)
	if err != nil {
		return nil, err
	}

	var batchResp BatchResponse
	if err := cp.codec.Decode(batchRespBytes, &batchResp); err != nil {
		return nil, err
	}

	if len(batchResp.Results) != 1 {
		return nil, Errf("unexpected batch results")
	}

	result := batchResp.Results[0]

	if result.MessageType == uint8(Msg.Error) {
		return nil, Err(result.Message)
	}

	responsePacket := Packet{
		Action:    packet.Action,
		HandlerID: packet.HandlerID,
		ReqID:     result.ReqID,
		Data:      result.Data,
	}

	return cp.codec.Encode(responsePacket)
}
