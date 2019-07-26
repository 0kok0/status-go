package protobuf

// IsDatasyncMessage checks whether there are any known field in the protobuf
// message
func (m *Payload) IsValid() bool {
	return len(m.Messages)+len(m.Acks)+len(m.Offers)+len(m.Requests) != 0
}
