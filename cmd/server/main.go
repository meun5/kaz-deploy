package main

import "github.com/meun5/kaz-deploy/kaz"

func main() {
	s := kaz.Server{
		Port:        8080,
		Address:     "127.0.0.1",
		ReleaseMode: kaz.Debug,
	}

	s.Run()
}
