package service

import (
	"context"
	"log"
	"os/exec"
	"sync"
	"time"
)

const (
	Start     int = 1
	Run       int = 2
	Final     int = 3
	Error     int = 4
	Interrupt int = 5
)

type Conns struct {
	sync.RWMutex
	list sync.Map
}

type Conn struct {
	Key string `json:"key"`

	Src          string    `json:"src"`
	Dst          string    `json:"dst"`
	Cmd          *exec.Cmd `json:"-"`
	sync.RWMutex `json:"-"`
	Status       int `json:"status"` // 1 create  2 Running 3 Final 4 error 5 Interrupt
}

type APIData struct {
	Key string `json:"key"`
	Src string `json:"src"`
	Dst string `json:"dst"`
}

func NewConns(ctx context.Context) *Conns {
	ret := &Conns{}
	go func() {
		for {
			select {
			case <-time.Tick(5 * time.Second):
				ret.list.Range(func(key, value interface{}) bool {
					item := value.(*Conn)
					if item.GetStatus() > 2 {
						ret.Remove(item.Key)
					}
					return true
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	return ret

}

func (c *Conns) Get(Key string) *Conn {
	if item, ok := c.list.Load(Key); !ok {
		return nil
	} else {
		return item.(*Conn)
	}
}

func (c *Conns) GetAll() []Conn {
	list := []Conn{}
	c.list.Range(func(key, value interface{}) bool {
		item := value.(*Conn)
		list = append(list, Conn{
			Key:    item.Key,
			Src:    item.Src,
			Dst:    item.Dst,
			Status: item.GetStatus(),
		})
		return true
	})
	return list
}

func (c *Conns) Add(item APIData) {
	c.Lock()
	defer c.Unlock()
	conn := c.Get(item.Key)
	cmd := exec.Command("./ffmpeg", "-readrate", "2", "-i", item.Src, "-codec", "copy", "-f", "flv", "-flvflags", "no_duration_filesize", item.Dst)
	if conn == nil {
		conn = &Conn{
			Key:    item.Key,
			Src:    item.Src,
			Dst:    item.Dst,
			Status: Start,
			Cmd:    cmd,
		}
		conn.Run()
		c.list.Store(item.Key, conn)

	} else {
		if conn.GetStatus() == Run {
			if item.Src != conn.Src || item.Dst != conn.Dst {
				conn.SetStatus(Interrupt)
			} else {
				return
			}
		}
		conn.Src = item.Src
		conn.Dst = item.Dst
		conn.Cmd = cmd
		conn.SetStatus(Start)
		conn.Run()
	}
}

func (c *Conns) Remove(Key string) {
	if item, ok := c.list.LoadAndDelete(Key); ok {
		log.Println("will be Removed Key:", Key)
		item.(*Conn).SetStatus(Interrupt)
	}
}

func (c *Conns) RemoveAll() {
	c.Lock()
	c.Unlock()
	c.list.Range(func(key, value interface{}) bool {
		value.(*Conn).SetStatus(Interrupt)
		c.list.Delete(key)
		return true
	})

}

func (c *Conn) GetStatus() int {
	c.RLock()
	defer c.RUnlock()
	return c.Status
}

func (c *Conn) SetStatus(phase int) {
	c.Lock()
	defer c.Unlock()

	if phase == Interrupt {
		if c.Status == Run {
			err := c.Cmd.Process.Kill()
			if err != nil {
				log.Println("[error]", "ffmpeg Kill Process :", err)
			}
		}
	}
	c.Status = phase
}

func (c *Conn) Run() {
	go func() {

		// ./ffmpeg -readrate 2 -i https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8 -codec copy -f flv -flvflags no_duration_filesize rtmp://domain/hls/abc
		// test source https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8
		//cmd := exec.Command("./ffmpeg", "-readrate", "2", "-i", c.Src, "-codec", "copy", "-f", "flv", "-flvflags", "no_duration_filesize", c.Dst)
		//不抛出duration_filesize警告

		//c.Cmd.Stdout = os.Stdout
		//c.Cmd.Stderr = os.Stderr

		if cmdStarterr := c.Cmd.Start(); cmdStarterr != nil {
			log.Println("[error]", "ffmpeg Command Start :", cmdStarterr)
			c.SetStatus(Error)
			return
		}
		log.Println("ffmpeg transfer :", c.Src, "->", c.Dst)
		c.SetStatus(Run)
		if waiterr := c.Cmd.Wait(); waiterr != nil {
			log.Println("[error]", "ffmpeg Command Wait :", waiterr)
			c.SetStatus(Error)
		}

		if c.GetStatus() == Run {
			c.SetStatus(Final)
		}

		log.Println("ffmpeg transfer done :", c.Src, "->", c.Dst)

	}()
}
