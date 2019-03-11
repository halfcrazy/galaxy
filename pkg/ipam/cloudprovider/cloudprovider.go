package cloudprovider

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"git.code.oa.com/gaiastack/galaxy/pkg/ipam/cloudprovider/rpc"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/util/trace"
)

type CloudProvider interface {
	AssignIP(in *rpc.AssignIPRequest) (*rpc.AssignIPReply, error)
	UnAssignIP(in *rpc.UnAssignIPRequest) (*rpc.UnAssignIPReply, error)
}

type GRPCCloudProvider struct {
	init              sync.Once
	cloudProviderAddr string
	client            rpc.IPProviderServiceClient
	backOff           wait.Backoff
	timeout           time.Duration
}

func NewGRPCCloudProvider(cloudProviderAddr string) CloudProvider {
	return &GRPCCloudProvider{
		backOff: wait.Backoff{
			Steps:    4,
			Duration: 10 * time.Millisecond,
			Factor:   5.0,
			Jitter:   0.1},
		timeout:           time.Second * 3,
		cloudProviderAddr: cloudProviderAddr,
	}
}

func (p *GRPCCloudProvider) connect() {
	p.init.Do(func() {
		glog.V(3).Infof("dial cloud provider with address %s", p.cloudProviderAddr)
		conn, err := grpc.Dial(p.cloudProviderAddr, grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("tcp", addr, timeout)
		}), grpc.WithInsecure())
		if err != nil {
			glog.Fatalf("failed to connect to cloud provider %s: %v", p.cloudProviderAddr, err)
		}
		p.client = rpc.NewIPProviderServiceClient(conn)
	})
}

func (p *GRPCCloudProvider) AssignIP(in *rpc.AssignIPRequest) (reply *rpc.AssignIPReply, err error) {
	p.connect()
	glog.V(3).Infof("AssignIP %v", in)
	t := trace.New("AssignIP")
	defer t.LogIfLong(time.Second)
	err = wait.ExponentialBackoff(p.backOff, func() (done bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		defer cancel()
		reply, err = p.client.AssignIP(ctx, in)
		glog.V(4).Infof("reply %v, err %v", reply, err)
		if err != nil {
			t.Step(fmt.Sprintf("AssignIP for %v failed: %v", in, err))
			return false, err
		}
		return true, nil
	})
	return
}

func (p *GRPCCloudProvider) UnAssignIP(in *rpc.UnAssignIPRequest) (reply *rpc.UnAssignIPReply, err error) {
	p.connect()
	glog.V(3).Infof("UnAssignIP %v", in)
	t := trace.New("UnAssignIP")
	defer t.LogIfLong(time.Second)
	err = wait.ExponentialBackoff(p.backOff, func() (done bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		defer cancel()
		reply, err = p.client.UnAssignIP(ctx, in)
		glog.V(4).Infof("reply %v, err %v", reply, err)
		if err != nil {
			// Expect cloud provider returns success if already unassigned
			t.Step(fmt.Sprintf("UnAssignIP for %+v failed: %v", in, err))
			return false, err
		}
		return true, nil
	})
	return
}
