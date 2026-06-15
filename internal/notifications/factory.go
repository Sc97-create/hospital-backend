package notifications

import "errors"

type Factory struct {
	senders map[ChannelType]Sender
}

func NewFactory(senders map[ChannelType]Sender) *Factory {
	return &Factory{senders: senders}
}
func (f *Factory) Get(channel ChannelType) (Sender, error) {
	sender, ok := f.senders[channel]
	if !ok {
		return nil, errors.New("unsupported channel")
	}

	return sender, nil
}
