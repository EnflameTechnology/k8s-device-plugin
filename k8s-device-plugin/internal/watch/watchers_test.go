// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package watch

import (
	"fmt"
	"syscall"
	"testing"

	. "github.com/agiledragon/gomonkey"
	"github.com/fsnotify/fsnotify"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewFSWatcher(t *testing.T) {
	Convey("test newFSWatcher\n", t, func() {
		Convey("1. test newFSWatcher success\n", func() {
			_, err := Files("./watchers.go")
			So(err, ShouldEqual, nil)
		})

		Convey("2. test NewWatcher failed\n", func() {
			errInfo := fmt.Errorf("NewWatcher failed")
			patcheListen := ApplyFunc(fsnotify.NewWatcher, func() (*fsnotify.Watcher, error) {
				return nil, errInfo
			})
			defer patcheListen.Reset()

			_, err := Files("./server.go")
			So(err, ShouldEqual, errInfo)
		})

		Convey("3. test newFSWatcher file not exist\n", func() {
			_, err := Files("./zxx.go")
			So(err.Error(), ShouldEqual, "no such file or directory")
		})
	})
}

func TestNewOSWatcher(t *testing.T) {
	Convey("test newOSWatcher\n", t, func() {
		Convey("1. test newOSWatcher success\n", func() {
			res := Signals(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
			So(len(res), ShouldEqual, 0)
		})
	})
}
