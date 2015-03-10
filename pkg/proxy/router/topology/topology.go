// Copyright 2014 Wandoujia Inc. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package topology

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/ngaut/zkhelper"

	"github.com/pcncadcache/cachesystempkg/models"

	"github.com/juju/errors"
	topo "github.com/ngaut/go-zookeeper/zk"
	log "github.com/ngaut/logging"
)

type TopoUpdate interface {
	OnGroupChange(groupId int)
	OnSlotChange(slotId int)
}

type ZkFactory func(zkAddr string) (zkhelper.Conn, error)

type Topology struct {
	ProductName string
	zkAddr      string
	zkConn      zkhelper.Conn
	fact        ZkFactory
}

func (top *Topology) GetGroup(groupId int) (*models.ServerGroup, error) {
	return models.GetGroup(top.zkConn, top.ProductName, groupId)
}

func (top *Topology) Exist(path string) (bool, error) {
	return zkhelper.NodeExists(top.zkConn, path)
}

func (top *Topology) GetSlotByIndex(i int) (*models.Slot, *models.ServerGroup, error) {
	slot, err := models.GetSlot(top.zkConn, top.ProductName, i)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	log.Debugf("get slot %d : %+v", i, slot)
	if slot.State.Status != models.SLOT_STATUS_ONLINE && slot.State.Status != models.SLOT_STATUS_MIGRATE {
		log.Errorf("slot not online, %+v", slot)
	}

	groupServer, err := models.GetGroup(top.zkConn, top.ProductName, slot.GroupId)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	return slot, groupServer, nil
}

func NewTopo(ProductName string, zkAddr string, f ZkFactory) *Topology {
	t := &Topology{zkAddr: zkAddr, ProductName: ProductName, fact: f}
	if t.fact == nil {
		t.fact = zkhelper.ConnectToZk
	}
	t.InitZkConn()
	return t
}

func (top *Topology) InitZkConn() {
	var err error
	top.zkConn, err = top.fact(top.zkAddr)
	if err != nil {
		log.Fatal(err)
	}
}

func (top *Topology) GetActionWithSeq(seq int64) (*models.Action, error) {
	return models.GetActionWithSeq(top.zkConn, top.ProductName, seq)
}

func (top *Topology) GetActionWithSeqObject(seq int64, act *models.Action) error {
	return models.GetActionObject(top.zkConn, top.ProductName, seq, act)
}

func (top *Topology) GetActionSeqList(productName string) ([]int, error) {
	return models.GetActionSeqList(top.zkConn, productName)
}

func (top *Topology) IsChildrenChangedEvent(e interface{}) bool {
	return e.(topo.Event).Type == topo.EventNodeChildrenChanged
}

func (top *Topology) CreateProxyInfo(pi *models.ProxyInfo) (string, error) {
	return models.CreateProxyInfo(top.zkConn, top.ProductName, pi)
}

func (top *Topology) CreateProxyFenceNode(pi *models.ProxyInfo) (string, error) {
	return models.CreateProxyFenceNode(top.zkConn, top.ProductName, pi)
}

func (top *Topology) GetProxyInfo(proxyName string) (*models.ProxyInfo, error) {
	return models.GetProxyInfo(top.zkConn, top.ProductName, proxyName)
}

func (top *Topology) GetActionResponsePath(seq int) string {
	return path.Join(models.GetWatchActionPath(top.ProductName), "action_"+fmt.Sprintf("%0.10d", seq))
}

func (top *Topology) SetProxyStatus(proxyName string, status string) error {
	return models.SetProxyStatus(top.zkConn, top.ProductName, proxyName, status)
}

func (top *Topology) Close(proxyName string) {
	// delete fence znode
	pi, err := models.GetProxyInfo(top.zkConn, top.ProductName, proxyName)
	if err != nil {
		log.Error("killing fence error, proxy %s is not exists", proxyName)
	} else {
		zkhelper.DeleteRecursive(top.zkConn, path.Join(models.GetProxyFencePath(top.ProductName), pi.Addr), -1)
	}
	// delete ephemeral znode
	zkhelper.DeleteRecursive(top.zkConn, path.Join(models.GetProxyPath(top.ProductName), proxyName), -1)
	top.zkConn.Close()
}

func (top *Topology) DoResponse(seq int, pi *models.ProxyInfo) error {
	//create response node
	actionPath := top.GetActionResponsePath(seq)
	//log.Debug("actionPath:", actionPath)
	data, err := json.Marshal(pi)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = top.zkConn.Create(path.Join(actionPath, pi.Id), data,
		0, zkhelper.DefaultACLs())

	return err
}

func (top *Topology) doWatch(evtch <-chan topo.Event, evtbus chan interface{}) {
	e := <-evtch
	log.Infof("topo event %+v", e)
	if e.State == topo.StateExpired {
		log.Fatalf("session expired: %+v", e)
	}

	switch e.Type {
	//case topo.EventNodeCreated:
	//case topo.EventNodeDataChanged:
	case topo.EventNodeChildrenChanged: //only care children changed
		//todo:get changed node and decode event
	default:
		log.Warningf("%+v", e)
	}

	evtbus <- e
}

func (top *Topology) WatchChildren(path string, evtbus chan interface{}) ([]string, error) {
	content, _, evtch, err := top.zkConn.ChildrenW(path)
	if err != nil {
		return nil, errors.Trace(err)
	}

	go top.doWatch(evtch, evtbus)
	return content, nil
}

func (top *Topology) WatchNode(path string, evtbus chan interface{}) ([]byte, error) {
	content, _, evtch, err := top.zkConn.GetW(path)
	if err != nil {
		return nil, errors.Trace(err)
	}

	go top.doWatch(evtch, evtbus)
	return content, nil
}
