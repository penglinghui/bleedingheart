package main

import (
	"fmt"
	"sync"
)

type Model struct {
	sync.RWMutex
	dir	string
	updated int64
	global	map[string]*BhFile
	local   map[string]*BhFile
	need	map[string]bool
}

func InitModel(dir string) {
	ensureDir(confDir)
	g_Model = &Model{
		dir:	dir,
		global: make(map[string]*BhFile),
		local:  make(map[string]*BhFile),
		need:	make(map[string]bool),
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

