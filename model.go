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
	updated int64		    // time from the master node for the global index timestamp
	global	map[string]*BhFile  // view of the master node
	local   map[string]*BhFile  // view of the local files
	need	map[string]bool
	activeFile io.WriterAt
	receivedBlock chan bool
}

func InitModel(dir string) {
	ensureDir(confDir)
	g_Model = &Model{
		dir:	dir,
		updated: 0,
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
		if g_IsMaster {
			m.updated = time.Now().Unix()
		}
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

func (m *Model) UpdateIndex(updated_new int64, fs []*BhFile) {
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
		m.updated = updated_new
		m.recomputeNeed()
	}
}

func (m *Model) recomputeNeed() {
	m.need = make(map[string]bool)
	for n, gf := range m.global {
		lf, ok := m.local[n]
		if ok && n != BytesToString(lf.Name) {
			panic("Corrupted map")
		}
		if !ok || *gf.Modified > *lf.Modified {
			m.need[n] = true
		}
	}
	fmt.Println(len(m.need), "files need update")
}

func (m *Model) UpdateLocal(f BhFile) {
	m.Lock()
	defer m.Unlock()

	if ef, ok := m.local[BytesToString(f.Name)]; !ok || ef.Modified != f.Modified {
		m.local[BytesToString(f.Name)] = &f
		m.recomputeNeed()
	}
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
		Hash: pmes.BlockData.Hash,
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
	var err error
	m.RLock()
	var localFile = m.local[name]
	var globalFile = m.global[name]
	m.RUnlock()

	if localFile == nil {
		localFile = &BhFile{
		}
	}

	filename := path.Join(m.dir, name)
	sdir := path.Dir(filename)

	_, err = os.Stat(sdir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(sdir, 0777)
	}

	tmpFilename := tempName(filename, *globalFile.Modified)

	// On Windows, rename only works after tmpFile is closed
	defer func() {
		if err == nil {
			for i:=0; i<10; i++ {
				err = os.Rename(tmpFilename, filename)
				if err == nil {
					break
				}
				fmt.Println("Rename failed. Retry...", i, err)
				time.Sleep(time.Duration(i+1)*time.Second)
			}

			fmt.Printf("Validated %s\n", filename)
		}
	}()

	tmpFile, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}
	m.activeFile = tmpFile
	defer func() {
		tmpFile.Close()
		fmt.Println("Closed tmpfile")
	}()

	_, remote := BlockList(localFile.Blocks).To(globalFile.Blocks)
	var fetchDone sync.WaitGroup

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
			<-m.receivedBlock
			fmt.Println("Received data")
		}
		fetchDone.Done()
	}()

	fetchDone.Wait()

	rf, err := os.Open(tmpFilename)
	if err != nil {
		return err
	}
	defer func() {
		rf.Close()
		fmt.Println("Closed file")
	}()

	writtenBlocks, err := Blocks(rf, BlockSize)
	if err != nil {
		return err
	}
	if len(writtenBlocks) != len(globalFile.Blocks) {
		return fmt.Errorf("%s: blocks %d != %d", tmpFilename, len(writtenBlocks), len(globalFile.Blocks))
	}
	for i := range writtenBlocks {
		if bytes.Compare(writtenBlocks[i].Hash, globalFile.Blocks[i].Hash) != 0 {
			err = fmt.Errorf("%s, hash mismatch after sync\n %v\n %v", tmpFilename, writtenBlocks[i], globalFile.Blocks[i])
			return err
		}
	}

	err = os.Chtimes(tmpFilename, time.Unix(*globalFile.Modified, 0), time.Unix(*globalFile.Modified, 0))
	if err != nil {
		return err
	}

	return nil
}

func (m *Model) puller() {
	if g_IsMaster {
		fmt.Println("Can't pull from master")
		return
	}
	done := false
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
				fmt.Println("Fully synced")
				done = true
				break
			}

			err := m.pullFile(n)
			if err == nil {
				m.UpdateLocal(f)
				fmt.Println(BytesToString(f.Name))
			} else {
				fmt.Println(err)
			}
		}
		if done {
			break
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

