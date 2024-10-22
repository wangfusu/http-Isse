package sse

import (
	"fmt"
	"http-Isse/backstream"
	"net/http"
	"testing"
	"time"
)

func TestChan(t *testing.T) {

	c := make(chan string, 10)

	c <- "1"
	close(c)
	//for ev := range c {
	//	fmt.Println(ev)
	//}
	////
	////go func() {
	////	for ev := range c {
	////		fmt.Println(ev)
	////	}
	////}()
	//
	//time.Sleep(1 * time.Second)
	//c <- "2"
	//
	close(c)
	close(c)
	time.Sleep(1 * time.Second)

	//
	//// 使用select语句来判断通道是否被关闭
	//select {
	//case c <- "42":
	//	fmt.Println("数据已成功发送到通道")
	//default:
	//	fmt.Println("无法发送数据，通道已关闭")
	//}

}

func TestSse(t *testing.T) {
	http.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		stream, err := CreateStream(request, writer)
		if err != nil {
			fmt.Println(err)
			return
		}
		go func() {
			for {
				time.Sleep(3 * time.Second)
				stream.SendEventData(&EventData{Data: []byte("111111"), Event: []byte("test")})
				stream.SendEventData(&EventData{Data: []byte("2222222")})
			}
		}()

		stream.OnClose(func() {
			fmt.Println("关闭")
		})

		//同步
		stream.SendEventData(&EventData{Data: []byte("33333"), Event: []byte("tongbu")})

		go func() {
			time.Sleep(20 * time.Second)
			fmt.Println("完成")
			stream.Done()
		}()

		stream.Wait()
		fmt.Println("退出")
		stream.Done()
		fmt.Println("再次推出")
		time.Sleep(100 * time.Second)
	})
	http.ListenAndServe(":8080", nil)
}

type TestNoNameI interface {
	Test() string
	TT()
}

func A(handle TestNoNameI) {
	handle.Test()
	handle.TT()
}

type Msg struct {
	ss string
}

func It() backstream.BSteam[*Msg] {
	stream := backstream.NewStream[*Msg]()
	stream.OnStop(func() {
		fmt.Println("停止")
	})
	go func() {
		time.Sleep(3 * time.Second)
		stream.Done()
	}()
	m := &Msg{ss: "test"}
	stream.Send(m)
	return stream
}

func TestSteam(t *testing.T) {
	it := It()
	go func() {
		for msg := range it.Messages() {
			fmt.Println(msg.ss)
		}
	}()

	time.Sleep(10 * time.Second)
	it.Stop()

}
