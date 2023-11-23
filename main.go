package main

import (
	"example/timesheet/router"
)

func main() {
	r := router.SetUpRouter()
	if err := r.Run("0.0.0.0:8080"); err != nil {
		panic("server stopped")
	}
}
