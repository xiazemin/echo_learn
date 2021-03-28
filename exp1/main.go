package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	mw "github.com/labstack/echo/middleware"
	"github.com/thoas/stats"
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var users map[string]user

func init() {
	users = map[string]user{
		"1": user{
			ID:   "1",
			Name: "Wreck-It Ralph",
		},
	}
}

func createUser(c echo.Context) error {
	u := new(user)
	if err := c.Bind(u); err == nil {
		users[u.ID] = *u
		return c.JSON(http.StatusCreated, u)
	}
	return nil
}

func getUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, users)
}

func getUser(c echo.Context) error {
	return c.JSON(http.StatusOK, users[c.Param("id")])
}

func main() {
	e := echo.New()

	//*************************//
	//   Built-in middleware   //
	//*************************//
	e.Use(mw.Logger())

	//****************************//
	//   Third-party middleware   //
	//****************************//
	// https://github.com/rs/cors
	/*
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{\"hello\": \"world\"}"))
		})
		handler := cors.Default().Handler(mux)

		e.Use(cors.Default().Handler(mux))
	*/

	// https://github.com/thoas/stats
	s := stats.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://foo.com", "http://test.com"},
		AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	}))
	// Route
	e.GET("/stats", func(c echo.Context) error {
		return c.JSON(200, s.Data())
	})

	// Serve index file
	e.Static("/index", "public/index.html")

	// Serve static files
	e.Static("/js", "public/js")

	//************//
	//   Routes   //
	//************//
	e.POST("/users", createUser)
	e.GET("/users", getUsers)
	e.GET("/users/:id", getUser)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
