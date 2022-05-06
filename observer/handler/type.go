package handler

type JobNameParam struct {
	Namespace string `uri:"namespace" json:"namespace" binding:"required"`
	Job       string `uri:"job" json:"job" binding:"required"`
}

type JobKeyParam struct {
	Key string `uri:"key" json:"key" binding:"required"`
}
