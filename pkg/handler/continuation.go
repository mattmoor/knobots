package handler

import "encoding/json"

type Continuation struct {
	Type   string
	Source string
	Data   []byte
}

func ToContinuation(r Response) (*Continuation, error) {
	// TODO(mattmoor): Factor a ToContinuation method.
	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return &Continuation{
		Type:   r.GetType(),
		Source: r.GetSource(),
		Data:   b,
	}, nil
}

func (c *Continuation) AsResponse() Response {
	return &continuation{c: c}
}

type continuation struct {
	c *Continuation
}

var _ Response = (*continuation)(nil)

func (c *continuation) GetSource() string {
	return c.c.Source
}

func (c *continuation) GetType() string {
	return c.c.Type
}

func (c *continuation) MarshalJSON() ([]byte, error) {
	return c.c.Data, nil
}
