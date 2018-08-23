package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"
	//ggio "github.com/gogo/protobuf/io"
	//ctxio "github.com/jbenet/go-context/io"
	//"github.com/libp2p/go-libp2p-net"
)

type Model struct {
	sync.RWMutex
	dir	string
	updated int64
	global	map[string]*BhFile
	local   map[string]*BhFile
	need	map[string]bool
	activeFile io.WriterAt
	receivedBlock chan bool
}

func InitModel(dir string) {
	ensureDir(confDir)
	g_Model = &Model{
		dir:	dir,
		global: make(map[string]*BhFile),
		local:  make(map[string]*BhFile),
		need:	make(map[string]bool),
		receivedBlock: make(chan bool),
	}
}

func (m *Model) LocalFile(name string) (*BhFile, bool) {
	m.RLock()
	defer m.RUnlock()
	f, ok := m.local[name]
	return f, ok
}

func (m *Model) ReplaceLocal(fs []*BhFile) {
	m.Lock()
	defer m.Unlock()

	var updated bool
	var newLocal = make(map[string]*BhFile)
	for _, f := range fs {
		fmt.Println("File:", BytesToString(f.Name))
		newLocal[BytesToString(f.Name)] = f
		ef := m.local[BytesToString(f.Name)]
		if ef != nil {
			if ef.Modified != f.Modified {
				updated = true
			}
		}
	}

	if len(newLocal) != len(m.local) {
		updated = true
	}

	if updated {
		fmt.Println("m.local updated")
		m.local = newLocal
		// go m.boradcastIndex()
	}
}

func (m *Model) GetLocalFiles() []*BhFile {
	m.RLock()
	defer m.RUnlock()

	var files []*BhFile
	for f := range m.local {
		files = append(files, m.local[f])
	}
	return files
}

func (m *Model) UpdateIndex(fs []*BhFile) {
	m.Lock()
	defer m.Unlock()

	var updated bool
	var newGlobal = make(map[string]*BhFile)
	for _, f := range fs {
		fmt.Println("File:", BytesToString(f.Name))
		newGlobal[BytesToString(f.Name)] = f
		ef := m.global[BytesToString(f.Name)]
		if ef != nil {
			if *ef.Modified != *f.Modified {
				fmt.Println("...modified ", *ef.Modified, " != ", *f.Modified);
				updated = true
			}
		}
	}

	if len(newGlobal) != len(m.global) {
		fmt.Println(len(newGlobal), "!=", len(m.global))
		updated = true
	}

	if updated {
		fmt.Println("m.global updated")
		m.global = newGlobal
		m.recomputeNeed()
	}
}

func (m *Model) recomputeNeed() {
	m.need = make(map[string]bool)
	for n, gf := range m.global {
		lf, ok := m.local[n]
		if !ok || *gf.Modified > *lf.Modified {
			m.need[n] = true
		}
	}
	fmt.Println(len(m.need), "files need update")
}

func (m *Model) RequestGlobal(name string, offset uint64, size uint32, hash []byte) error {

	if g_IsMaster {
		return errors.New("Unexpected RequestGlobal from master")
	}

	t := BhMessage_BH_REQUEST
	pmes := &BhMessage {
		Type: &t,
	}
	bd := new (BhBlockData)
	bd.Name = StringToBytes(name)
	bd.Offset = &offset
	bd.Length= &size
	bd.Hash = hash
	pmes.BlockData = bd
	if err := g_StreamManager.SendMessage(g_MasterID, pmes); err != nil {
		return err
	}

	return nil
}

func (m *Model) BuildResponse(pmes *BhMessage, rpmes *BhMessage) error {
	if !g_IsMaster {
		return errors.New("Currently only master is able to return response")
	}
	if pmes.BlockData == nil {
		return errors.New("Invalid request")
	}
	if BlockSize < *pmes.BlockData.Length {
		return errors.New("Invalid block size")
	}

	bd := &BhBlockData{
		Name: pmes.BlockData.Name,
		Offset: pmes.BlockData.Offset,
		Length: pmes.BlockData.Length,
	}
	fn := path.Join(m.dir, BytesToString(bd.Name))
	fmt.Printf("Building response for %s, offset %d, length %d", fn, *bd.Offset, *bd.Length)
	fd, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer fd.Close()
	bd.Data = make([]byte, int(*bd.Length))
	_, err = fd.ReadAt(bd.Data, int64(*bd.Offset))
	if err != nil {
		return err
	}
	rpmes.BlockData = bd
	return nil
}

type content struct {
	offset uint64
	data []byte
}

var buffers = make(chan []byte, 32)

func GetBuffer(size int) []byte {
	var buf []byte
	select {
	case buf = <-buffers:
	default:
	}
	if len(buf) < size {
		return make([]byte, size)
	}
	return buf[:size]
}

func PutBuffer(buf []byte) {
	if cap(buf) == 0 {
		return
	}
	buf = buf[:cap(buf)]
	select {
	case buffers <- buf:
	default:
	}
}

func applyContent(cc <-chan content, dst io.WriterAt) error {
	var err error

	for c := range cc {
		_, err = dst.WriteAt(c.data, (int64)(c.offset))
		if err != nil {
			return err
		}
		PutBuffer(c.data)
	}

	return nil
}

func (m *Model) WriteBlock(b *BhBlockData) error {
	if b == nil {
		m.receivedBlock <- false
		return errors.New("Nil blockdata")
	}
	if len(b.Data) != int(*b.Length) {
		return errors.New("Mismatched data length")
	}
	if *b.Length > BlockSize {
		return errors.New("Wrong blocksize")
	}
	fmt.Println("WriteBlock to active file")
	m.receivedBlock <- true
	_,err := m.activeFile.WriteAt(b.Data, (int64)(*b.Offset))
	return err
}

func (m *Model) pullFile(name string) error {
	m.RLock()
	var localFile = m.local[name]
	var globalFile = m.global[name]
	m.RUnlock()

	filename := path.Join(m.dir, name)
	sdir := path.Dir(filename)

	_, err := os.Stat(sdir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(sdir, 0777)
	}

	tmpFilename := tempName(filename, *globalFile.Modified)
	tmpFile, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}
	defer tmpFile.Close()

	m.activeFile = tmpFile

	contentChan := make(chan content, 32)
	var applyDone sync.WaitGroup
	applyDone.Add(1)
	go func() {
		applyContent(contentChan, tmpFile)
		applyDone.Done()
	}()

	local, remote := BlockList(localFile.Blocks).To(globalFile.Blocks)
	fmt.Println(len(local))
	var fetchDone sync.WaitGroup

	/*
	fetchDone.Add(1)
	go func() {
		for _, block := range local {
			data, err := m.Request("<local>", name, *block.Offset, *block.Length, block.Hash)
			if err != nil {
				break
			}
			contentChan <- content{
				offset: *block.Offset
				data: data,
			}
		}
		fetchDone.Done()
	}()
	*/

	var remoteBlocksChan = make(chan BhBlock)
	go func() {
		for _, block := range remote {
			remoteBlocksChan <- *block
		}
		close(remoteBlocksChan)
	}()

	fetchDone.Add(1)
	go func() {
		for block := range remoteBlocksChan {
			fmt.Println("Requesting data @", *block.Offset, name)
			err := m.RequestGlobal(name, *block.Offset, *block.Length, block.Hash)
			if err != nil {
				break
			}
			fmt.Println("Request sent, waiting for data")
			select {
			case <-m.receivedBlock:
				fmt.Println("Received data")
			}
		}
		fetchDone.Done()
	}()

	fetchDone.Wait()
	close(contentChan)
	applyDone.Wait()

	rf, err := os.Open(tmpFilename)
	if err != nil {
		return err
	}
	defer rf.Close()

	writtenBlocks, err := Blocks(rf, BlockSize)
	if err != nil {
		return err
	}
	if len(writtenBlocks) != len(globalFile.Blocks) {
		return fmt.Errorf("%s: blocks %d != %d", tmpFilename, len(writtenBlocks), len(globalFile.Blocks))
	}
	for i := range writtenBlocks {
		if bytes.Compare(writtenBlocks[i].Hash, globalFile.Blocks[i].Hash) != 0 {
			return fmt.Errorf("%s, hash mismatch after sync\n %v\n %v", tmpFilename, writtenBlocks[i], globalFile.Blocks[i])
		}
	}

	err = os.Chtimes(tmpFilename, time.Unix(*globalFile.Modified, 0), time.Unix(*globalFile.Modified, 0))
	if err != nil {
		return err
	}

	err = os.Rename(tmpFilename, filename)
	if err != nil {
		return err
	}

	return nil
}

func (m *Model) puller() {
	for {
		for {
			var n string
			var f BhFile

			m.RLock()
			for n = range m.need {
				break
			}
			if len(n) != 0 {
				f = *m.global[n]
			}
			m.RUnlock()

			if len(n) == 0 {
				break
			}

			err := m.pullFile(n)
			if err == nil {
				//m.UpdateLocal(f)
				fmt.Println(f.Name)
			} else {
				fmt.Println(err)
			}
		}
		time.Sleep(time.Second)
	}
}

func (m *Model) Dump() {
	m.RLock()
	defer m.RUnlock()

	fmt.Println("---------------- global ----------------")
	for f := range m.global {
		m.global[f].Dump()
	}
	fmt.Println("---------------- local ----------------")
	for f := range m.local {
		m.local[f].Dump()
	}
}

func (m *Model) Refresh() {
	fmt.Println("Refresh() ...")
	files := Walk()
	m.ReplaceLocal(files)
}

