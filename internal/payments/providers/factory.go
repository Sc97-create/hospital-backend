package providers

import "errors"

type IPaymentFactory interface {
	GetProvider(provider string) (Provider, error)
}
type PaymentFactory struct {
	senders map[string]Provider
}

func NewPaymentFactory(gateways ...Provider) *PaymentFactory {
	f := &PaymentFactory{
		senders: make(map[string]Provider),
	}
	for _, each := range gateways {
		f.senders[each.Name()] = each
	}
	return f

}

func (f *PaymentFactory) GetProvider(provider string) (Provider, error) {
	sender, ok := f.senders[provider]
	if !ok {
		return nil, errors.New("unsupported channel")
	}

	return sender, nil
}
