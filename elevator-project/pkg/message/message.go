package message

//Defines the JSON structures for your different message types (state updates, orders, heartbeats, ACKs)
//and functions to marshal/unmarshal these messages.
import (
	"encoding/json"
	"errors"
	"time"
)

type Message struct {
    Type      string          `json:"type"`
    Seq       uint64          `json:"seq"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   json.RawMessage `json:"payload"`
}

// MarshalMessage converts a Message struct into a JSON-formatted byte slice.
// This function is useful when you need to send the message over the network.
func MarshalMessage(msg Message) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// UnmarshalMessage converts a JSON-formatted byte slice into a Message struct.
// It returns the parsed Message and any error encountered during unmarshaling.
func UnmarshalMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return Message{}, err
	}
	return msg, nil
}

// ValidateMessage checks the integrity of a Message.
// It ensures that required fields are populated and, if a previous sequence number is provided,
// that the current message's sequence number is greater than the previous one.
// The prevSeq parameter should be set to 0 if no previous sequence exists.
func ValidateMessage(msg Message, prevSeq uint64) error {
	// Ensure the message type is not empty.
	if msg.Type == "" {
		return errors.New("message type is empty")
	}

	// Ensure the sequence number is not zero.
	// (Depending on your design, you might reserve zero as an invalid value.)
	if msg.Seq == 0 {
		return errors.New("message sequence number is zero")
	}

	// If a previous sequence number is provided, check that the new message's sequence is higher.
	if prevSeq > 0 && msg.Seq <= prevSeq {
		return errors.New("message sequence number is not greater than previous sequence number")
	}

	// Check that the timestamp is not in the future relative to the current time.
	if msg.Timestamp.After(time.Now()) {
		return errors.New("message timestamp is in the future")
	}

	return nil
}
