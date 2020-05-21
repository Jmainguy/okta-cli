package main

type oktaUsers []struct {
	Email  string   `yaml:"email"`
	First  string   `yaml:"first"`
	Groups []string `yaml:"groups"`
	Last   string   `yaml:"last"`
}
