package main

import (
	"github.com/zsdevX/DarkEye/common"
	"github.com/zsdevX/DarkEye/superscan/plugins"
	"golang.org/x/time/rate"
	"strconv"
	"sync"
	"time"
)

func New(ip string) *Scan {
	return &Scan{
		Ip:                     ip,
		ActivePort:             "80",
		PortRange:              common.PortList,
		Callback:               callback,
		BarCallback:            barCallback,
		ThreadNumber:           200,
		BarDescriptionCallback: descCallback,
	}
}

func (s *Scan) Run() {
	s.preCheck()
	fromTo, tot := common.GetPortRange(s.PortRange)
	taskAlloc := make(chan int, s.ThreadNumber)
	wg := sync.WaitGroup{}
	wg.Add(tot)
	for _, p := range fromTo {
		for p.From <= p.To {
			taskAlloc <- 1
			go func(port int) {
				for {
					s.Check(port)
					if !s.IsFireWallNotForbidden() {
						//被防火墙策略限制探测，等待恢复期（恢复期比较傻，需要优化）。
						time.Sleep(time.Second * 10)
						//恢复后从中断的端口重新检测
						continue
					}
					break
				}
				<-taskAlloc
				wg.Done()
			}(p.From)
			p.From++
		}
	}
	wg.Wait()
}

func (s *Scan) Check(p int) {
	defer func() {
		s.BarCallback(1)
	}()
	plg := plugins.Plugins{
		TargetIp:     s.Ip,
		TargetPort:   strconv.Itoa(p),
		TimeOut:      s.TimeOut,
		PortOpened:   false,
		NoTrust:      s.NoTrust,
		Worker:       s.PluginWorker,
		RateLimiter:  s.Rate,
		DescCallback: s.BarDescriptionCallback,
		RateWait:     rateWait,
	}
	plg.Check()
	if !plg.PortOpened {
		return
	}
	s.Callback(&plg)
}

func (s *Scan) preCheck() {
	plg := plugins.Plugins{
		TargetIp:    s.Ip,
		RateLimiter: mPps,
		RateWait:    rateWait,
		TimeOut:     s.TimeOut,
	}
	plg.PreCheck()
}

func (s *Scan) IsFireWallNotForbidden() bool {
	//为0不矫正
	if s.ActivePort == "0" {
		return true
	}
	maxRetries := 3
	for maxRetries > 0 {
		if common.IsAlive(s.Ip, s.ActivePort, s.TimeOut) == common.Alive {
			return true
		}
		maxRetries --
	}
	return false
}

func rateWait(r *rate.Limiter) {
	if r == nil {
		return
	}
	for {
		if r.Allow() {
			break
		} else {
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func callback(a interface{}) {
	//todo
}

func barCallback(i int) {
	//todo
}

func descCallback(i string) {
	//todo
}
