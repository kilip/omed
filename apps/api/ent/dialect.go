package ent

// Dialect returns the driver dialect name for the client.
func (c *Client) Dialect() string {
	return c.driver.Dialect()
}

// Dialect returns the driver dialect name for the transaction.
func (tx *Tx) Dialect() string {
	return tx.driver.Dialect()
}
