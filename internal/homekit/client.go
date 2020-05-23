package homekit

const ()

type Client interface {
}

type client struct {
	pin string
}

func NewClient() Client {
	return &client{
		pin: "25174512",
	}
}

func (c *client) TestFuelGauge() {

}
