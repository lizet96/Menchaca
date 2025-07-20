package handlers

type BodyResponse struct {
	IntCode string        `json:"intCode"`
	Data    []interface{} `json:"data"`
}

type StandardResponse struct {
	StatusCode int          `json:"statusCode"`
	Body       BodyResponse `json:"body"`
}
