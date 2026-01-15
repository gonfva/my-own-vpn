package credentials

// AWSCredentials holds AWS access credentials
type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// HetznerCredentials holds Hetzner API credentials
type HetznerCredentials struct {
	APIToken string
}

// Credentials is a union of all credential types
type Credentials struct {
	AWS     *AWSCredentials
	Hetzner *HetznerCredentials
}

// IsEmpty returns true if no credentials are set
func (c *AWSCredentials) IsEmpty() bool {
	return c == nil || (c.AccessKeyID == "" && c.SecretAccessKey == "")
}

// IsEmpty returns true if no credentials are set
func (c *HetznerCredentials) IsEmpty() bool {
	return c == nil || c.APIToken == ""
}
