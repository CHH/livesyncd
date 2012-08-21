package main

import(
	"github.com/howeyc/fsnotify"
)

const (
	SYNC_PUT    = 1
	SYNC_DELETE = 2
	SYNC_RENAME = 3
)

type SyncEvent struct {
	Name      string              // File name
	OldName   string              // Old name of the file, in case it was renamed.
	Type      int                 // Type of the sync event
	FileEvent *fsnotify.FileEvent // original fsnotify event
	Dir bool
}

func (self *SyncEvent) IsPut() bool {
	return self.Type == SYNC_PUT
}

func (self *SyncEvent) IsDelete() bool {
	return self.Type == SYNC_DELETE
}

func (self *SyncEvent) IsRename() bool {
	return self.Type == SYNC_RENAME
}
