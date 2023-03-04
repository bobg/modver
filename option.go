package modver

type Option func(*Comparer)

func Report(r bool) Option {
	return func(c *Comparer) {
		c.report = r
	}
}
