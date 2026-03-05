package rule

type Rule struct {
	ID      string `yaml:"id"`
	Listen  string `yaml:"listen"`
	Forward string `yaml:"forward"`
}

