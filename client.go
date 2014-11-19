package openvpn

type Client struct {
	CommonName       string
	PublicIP         string
	PrivateIP        string
	BytesRecived     string
	BytesSent        string
	LastRef          string
	waitForPrivateIP chan bool
	missing          int
	Env              map[string]string
}
