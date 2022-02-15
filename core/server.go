package core

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/streamingfast/dummy-blockchain/types"
)

const (
	homePage = `
<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/3.4.1/css/bootstrap.min.css" integrity="sha384-HSMxcRTRxnN+Bdg0JdbxYKrThecOKuH5zCYotlSAcp1+c8xmyTe9GYg1l9a69psu" crossorigin="anonymous">
<div class="container">
	<h1>Dummy Blockchain</h1>
	<p>You're looking at the Dummy block chain server implementation!</p>
	<hr/>
	<h2>Routes</h3>
	<ul>
		<li><code>/</code> - View current page</li>
		<li><code>/status</code> - Chain status</li>
		<li><code>/block</code> - Current block</li>
		<li><code>/blocks/:height</code> - Get block by height</li>
	</ul>
</div>
	`
)

type Server struct {
	*gin.Engine

	store *Store
	addr  string
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func NewServer(store *Store, addr string) Server {
	server := Server{
		Engine: gin.Default(),
		store:  store,
		addr:   addr,
	}

	server.GET("/", server.getHome)
	server.GET("/status", server.getStatus)
	server.GET("/block", server.getBlock)
	server.GET("/blocks/:id", server.getBlock)

	return server
}

func (s *Server) Start() error {
	logrus.WithField("addr", s.addr).Info("starting server")
	err := s.Run(s.addr)
	if err != nil {
		logrus.WithError(err).Error("cant start server")
	}

	return err
}

func (s *Server) getHome(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(200, homePage)
}

func (s *Server) getStatus(c *gin.Context) {
	c.JSON(200, s.store.meta)
}

func (s *Server) getBlock(c *gin.Context) {
	var (
		block *types.Block
		err   error
	)

	if id := c.Param("id"); len(id) > 0 {
		blockNum, err := strconv.Atoi(id)
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
			return
		}

		block, err = s.store.ReadBlock(uint64(blockNum))
	} else {
		block, err = s.store.CurrentBlock()
	}

	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	if block == nil {
		c.JSON(404, gin.H{"error": "block not found"})
		return
	}

	c.JSON(200, block)
}
