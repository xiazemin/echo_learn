package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
)

type User struct {
	Name  string `param:"name" query:"name" form:"name" json:"name" xml:"name` //curl -XGET http://localhost:1323/users/Joe\?email\=joe_email
	Email string `json:"email" form:"email" query:"email"`
}

func main() {
	e := echo.New()
	e.GET("/users/:name", func(c echo.Context) error {
		u := new(User)
		u.Name = c.Param("name")
		if err := c.Bind(u); err != nil {
			return c.JSON(http.StatusBadRequest, nil)
		}
		return c.JSON(http.StatusOK, u)
	})
	fmt.Println(e.Start(":1336"))
}
