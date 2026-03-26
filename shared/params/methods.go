package params

import (
	"errors"
	"time"
)

func (p *Payload) Getstring(key string) (value string, err error) {
	if val, ok := p.Param[key]; ok {
		if s, ok := val.(string); ok {
			return s, nil
		}

	}
	err = errors.New("not a string type")
	return "", err
}
func (p *Payload) GetObject(key string) (*Payload, error) {
	if val, ok := p.Param[key]; ok {
		if s, ok := val.(map[string]any); ok {
			return &Payload{Param: s}, nil
		}
	}
	err := errors.New("not a valid type")
	return nil, err
}
func (p *Payload) GetChildren(key string) ([]*Payload, error) {
	childrens := []*Payload{}
	if val, ok := p.Param[key]; ok {
		if parent, ok := val.([]any); ok {
			for _, each := range parent {
				if m, ok := each.(map[string]any); ok {
					childrens = append(childrens, &Payload{Param: m})
				}
			}
		}
		return childrens, nil
	}
	err := errors.New("not a valid type")
	return nil, err
}
func (p *Payload) GetStringArray(key string) ([]string, error) {
	if val, ok := p.Param[key]; ok {
		if s, ok := val.([]string); ok {
			return s, nil
		}
	}
	err := errors.New("not a valid type")
	return nil, err
}

// float, object,arrayobject,arraystring,arrayfloat
func (p *Payload) Getint(key string) (int, error) {
	if val, ok := p.Param[key]; ok {
		switch v := val.(type) {
		case float64:
			// JSON numbers are parsed as float64, so we need to convert
			return int(v), nil
		case int:
			return v, nil
		case int64:
			return int(v), nil
		default:
			return 0, errors.New("not a valid int type")
		}
	}
	return 0, errors.New("key not found")
}

func (p *Payload) Getfloat(key string) (float64, error) {
	if val, ok := p.Param[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return 0, errors.New("not a valid float type")
		}
	}
	return 0, errors.New("key not found")
}
func (p *Payload) GetBool(key string) (bool, error) {
	if val, ok := p.Param[key]; ok {
		if p, ok := val.(bool); ok {
			return p, nil
		}
	}
	err := errors.New("not valid type")
	return false, err
}
func (p *Payload) GetTime(key string) (time.Time, error) {
	if val, ok := p.Param[key]; ok {
		if p, ok := val.(time.Time); ok {
			return p, nil
		}
	}
	return time.Time{}, nil
}
func (p *Payload) GetInt64(key string) (int64, error) {
	if val, ok := p.Param[key]; ok {
		if p, ok := val.(int64); ok {
			return p, nil
		}
	}
	return 0, nil
}
