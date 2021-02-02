package main

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
	"net/http"
	"log"
	"github.com/gin-gonic/gin"

)

type PreparePutPeerRequest struct {
	PeerID     string  `json:"peer_id" binding:"required"`
	Addr       string  `json:"addr" binding:"required"`
	Credential string  `json:"credential" binding:"required"`
	Longitude  float64 `json:"longitude" binding:"required"`
	Latitude   float64 `json:"latitude" binding:"required"`
}

type LookupPeerRequest struct {
	Longitude  float64 `json:"longitude" binding:"required"`
	Latitude   float64 `json:"latitude" binding:"required"`
	Radius     float64 `json:"radius" binding:"required"`
}

func main(){
	app := gin.Default()
	register(app)
	srv := &http.Server{
		Addr: ":8000",
		Handler: app,
	}
	go func(){
		if err := srv.ListenAndServe(); err != http.ErrServerClosed{
			log.Fatalln("Server closed with error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	log.Printf("SIGNAL %d received, then shutting down...\n",<-quit)
	ctx, cancel := context.WithTimeout(context.Background(),30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil{
		log.Println("Failed to gracefully shutdown:",err)
	}
	log.Println("Server shutdown")
}

func register(e *gin.Engine){
	api := e.Group("/api")
	{
		api.POST("/peer",PreparePutPeer)
		api.GET("/peer/read",ReadPeer)
	}
}

func PreparePutPeer(c *gin.Context){
	req := &PreparePutPeerRequest{}
	if err := c.BindJSON(req); err != nil{
		log.Fatal(err)
		c.JSON(400,gin.H{"message": "invalid json of PreparePutPeer"})
		return
	}
	if err := preparePutPeer(req); err != nil{
		log.Fatal(err)
		c.JSON(500,gin.H{"message": "Failed insert peer information"})
	}
	c.JSON(200,gin.H{"message": "insert OK"})
}

func preparePutPeer(req *PreparePutPeerRequest)error{
	ctx := context.Background()
	if err := Insert(ctx,req); err != nil{
		return err
	}
	return nil
}

func ReadPeer(c *gin.Context){
	req := &LookupPeerRequest{}

	if err := c.BindJSON(req); err != nil{
		log.Fatal(err)
		c.JSON(400,gin.H{"message": "invalid json of ReadPeer"})
		return
	}
	if err := readPeer(c,req); err != nil{
		log.Fatal(err)
		c.JSON(500,gin.H{"message": "Failed read peer information"})
	}
	c.JSON(200,gin.H{"message": "read complete"})
}

func readPeer(c *gin.Context, req *LookupPeerRequest) error{
	db := Mysql()
	defer db.Close()
	rows, err := db.Query("SELECT peer_id,ST_X(location),ST_Y(location) from peer")
	if err != nil{
		return err
	}
	defer rows.Close()
	var loc []string
	for rows.Next(){
		var lng float64
		var lat float64
		var  peer string
		err := rows.Scan(&peer,&lng,&lat)
		//fmt.Println(lng,lat)
		if err != nil{
			return err
		}
		if LookupPeer(db,req,lng,lat){
			loc = append(loc,peer)
			//fmt.Println(peer)
		}
	}
	c.JSON(200,gin.H{"location": loc})
	return nil
}

func isContains(loc []string,peer string) bool{
	for _,v := range loc{
		if peer == v{
			return false
		}
	}
	return true
}


func Mysql() *sql.DB{
	url := "root:root@tcp(127.0.0.1:3306)/sample_db"
	db, err := sql.Open("mysql",url)
	if err != nil {
		log.Fatal(err)
	}
	if db == nil{
		log.Fatalf("Failed connect to mysql, url=%s",url)
	}
	return db
}

func Insert(ctx context.Context, req *PreparePutPeerRequest) error{
	db := Mysql()
	defer db.Close()
	exists, err := isExistPeer(req.PeerID,db)
	if err != nil {
		return err
	}
	if exists{
		return nil
	}
	statement := fmt.Sprintf("INSERT INTO peer(peer_id, addr, credential, location) VALUES (?, ?, ?, ST_GeomFromText(?))")
	pointValue := fmt.Sprintf(`POINT(%f %f)`, req.Longitude, req.Latitude)
	ins, err := db.Prepare(statement)
		if err != nil {
			return err
		}
		_, err = ins.Exec(req.PeerID, "dummy-addr", "dummy-credential", pointValue)
		if err != nil {
			return err
		}
		return nil
}

func LookupPeer(db *sql.DB, req *LookupPeerRequest,lng float64,lat float64)bool{
	query := "SELECT peer_id FROM peer WHERE ST_Within(ST_GeomFromText(?),ST_Buffer(POINT(?,?),?))"
	pointValue1 := fmt.Sprintf(`POINT(%f %f)`,lng,lat)
	radius := 0.009/1000*req.Radius
	var peer string
	err := db.QueryRow(query,pointValue1,req.Longitude,req.Latitude,radius).Scan(&peer)
	if err == sql.ErrNoRows{
		return false
	}else if err !=nil{
		log.Fatal(err)
		return false
	}
	return true
}

func isExistPeer(peerID string,db *sql.DB) (bool,error){
	var exists bool
	query := "SELECT exists (SELECT peer_id from peer where peer_id = ?)"
	err := db.QueryRow(query,peerID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows{
		return false, err
	}
	return exists,nil
}
