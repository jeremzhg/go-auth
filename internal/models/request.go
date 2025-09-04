package models

type Request struct{
	Subject string `json:"subject"` 
	Action string `json:"action"`
	Object string `json:"object"`
}