package types

type (
	Response struct {
		Data string `json:"data"`
		Meta meta   `json:"meta"`
	}
)
