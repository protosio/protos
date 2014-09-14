package main

import (
        //"database/sql"
        "fmt"
        //_ "github.com/mattn/go-sqlite3"
        "os"
        "os/exec"
        //"net/http"
        //"log"
	//"bytes"
	//"encoding/json"
	//"github.com/gorilla/mux"
        //"github.com/fsouza/go-dockerclient"
	//dkr "github.com/dotcloud/docker"
        "github.com/codegangsta/cli"
)

//func db() {
//
//        os.Remove("db/egor.db")
//
//        db, err := sql.Open("sqlite3", "db/egor.db")
//        if err != nil {
//                log.Fatal(err)
//        }
//        defer db.Close()
//
//        sql := `
//        create table foo (id integer not null primary key autoincrement, name text);
//        delete from foo;
//        `
//        _, err = db.Exec(sql)
//        if err != nil {
//                log.Printf("%q: %s\n", err, sql)
//                return
//        }
//
//        tx, err := db.Begin()
//        if err != nil {
//                log.Fatal(err)
//        }
//        stmt, err := tx.Prepare("insert into foo(name) values(?)")
//        if err != nil {
//                log.Fatal(err)
//        }
//        defer stmt.Close()
//        for i := 0; i < 5; i++ {
//                _, err = stmt.Exec(fmt.Sprintf("Alex %03d", i))
//                if err != nil {
//                        log.Fatal(err)
//                }
//        }
//        tx.Commit()
//
//        rows, err := db.Query("select id, name from foo")
//        if err != nil {
//                log.Fatal(err)
//        }
//        defer rows.Close()
//        fmt.Println("printing results")
//        for rows.Next() {
//                var id int
//                var name string
//                rows.Scan(&id, &name)
//                fmt.Println("---------")
//                fmt.Println(id, name)
//        }
//        rows.Close()
//
//        stmt, err = db.Prepare("select name from foo where id = ?")
//        if err != nil {
//                log.Fatal(err)
//        }
//        defer stmt.Close()
//        var name string
//        err = stmt.QueryRow("3").Scan(&name)
//        if err != nil {
//                log.Fatal(err)
//        }
//        fmt.Println(name)
//
//        //_, err = db.Exec("delete from foo")
//        //if err != nil {
//        //        log.Fatal(err)
//        //}
//
//        _, err = db.Exec("insert into foo(name) values('foo')")
//        if err != nil {
//                log.Fatal(err)
//        }
//
//        rows, err = db.Query("select id, name from foo")
//        if err != nil {
//                log.Fatal(err)
//        }
//        defer rows.Close()
//        for rows.Next() {
//                var id int
//                var name string
//                rows.Scan(&id, &name)
//                fmt.Println(id, name)
//        }
//}
//
//type User struct {
//	Email string
//	FirstName string
//}
//
//func GetUserHandler(w http.ResponseWriter, req *http.Request) {
//
//	user := User{ Email : "m...@me.com", FirstName: "Me"}
//
//	fmt.Println("GetUserHandler(): user: " + user.Email)
//	b, err := json.Marshal(user)
//	if err != nil {
//		fmt.Println(err)
//			w.WriteHeader(400)
//	} else {
//		w.Write(b)
//	}
//
// 	return 
//}
//
//func websrv() {
//
//    //r := mux.NewRouter()
//    //r.HandleFunc("/rest/user/", GetUserHandler).Methods("GET")
//    //r.Handle("/", http.FileServer(http.Dir("./")))
//
//    r := mux.NewRouter()
//    r.HandleFunc("/rest/user/", GetUserHandler).Methods("GET")
//    r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
//
//    // http.Handle("/rest/", r)
//    // http.Handle("/", http.FileServer(http.Dir("./")))
//
//    log.Fatal(http.ListenAndServe(":9009", r))
//}
//
//func test_docker() {
//        client, err := docker.NewClient("http://localhost:9196")
//        if err != nil {
//                log.Println("we haz error")
//                log.Fatal(err)
//        }	
//        fmt.Println(client)
//
//        id := "0b42a890ad91"
//        err = client.StartContainer(id, &dkr.HostConfig{})
//        if err != nil {
//                log.Fatal(err)
//        }
//       
//
//        opts := docker.ListContainersOptions{}
//
//        containers, err := client.ListContainers(opts)
//        if err != nil {
//                log.Fatal(err)
//        }
//        
//        fmt.Println(containers)
//
//        images, err := client.ListImages(true)
//        if err != nil {
//                log.Fatal(err)
//        }
//        fmt.Println(images)
//
//        
//}

func main() {
	//db()
        //test_docker()
        //websrv()

  app := cli.NewApp()
  app.Name = "egor"
  app.Usage = "iz good for your privacy"
  app.Action = func(c *cli.Context) {
    println("I work!")
  }

  app.Commands = []cli.Command{
  {
    Name:      "start",
    ShortName: "s",
    Usage:     "starts an application",
    Action: func(c *cli.Context) {
      println("Running application: ", c.Args().First())
      out, err := exec.Command("sh","-c",fmt.Sprintf("./images/%s/start.sh",c.Args().First())).Output()
      if err != nil {
        fmt.Printf("%s", err)
      }
      fmt.Printf("%s", out)
    },
  }}

  app.Run(os.Args)
}



