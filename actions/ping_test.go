package actions

func (as *ActionSuite) Test_PingPong() {
	res := as.JSON("/v1/rds/ping").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "pong")
}
