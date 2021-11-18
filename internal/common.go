package internal

import "log"

func check(err error) {
	if err != nil {
		log.Fatalf("Something went wrong: %q", err)
	}
}
