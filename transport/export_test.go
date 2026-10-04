package transport

// NewFakeMQTT returns an MQTT transport whose client is a fake device that
// answers every published frame, and a function returning the frames it got.
func NewFakeMQTT(deviceID string) (m *MQTT, frames func() [][]byte) {
	m = NewMQTT("tcp://192.0.2.1:1883", deviceID)
	client := &droppableClient{mockClient: mockClient{connected: true}, transport: m}
	m.client = client
	return m, client.published
}
