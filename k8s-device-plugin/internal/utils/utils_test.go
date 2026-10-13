// Copyright (c) 2024, ENFLAME INC.  All rights reserved.

package utils

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileIsExist(t *testing.T) {
	Convey("test FileIsExist\n", t, func() {
		Convey("1. test file exist\n", func() {
			file := "./utils.go"
			exist := FileIsExist(file)
			So(exist, ShouldEqual, true)
		})

		Convey("2. test file not exist\n", func() {
			file := "./utils1.go"
			exist := FileIsExist(file)
			So(exist, ShouldEqual, false)
		})
	})
}

func TestIntListContainValue(t *testing.T) {
	Convey("test IntList.ContainValue\n", t, func() {
		Convey("1. test not contain value\n", func() {
			intList := []int{1, 2, 3}
			ok := IntList(intList).ContainValue(4)
			So(ok, ShouldEqual, false)
		})

		Convey("2. test contain value\n", func() {
			intList := []int{1, 2, 3}
			ok := IntList(intList).ContainValue(2)
			So(ok, ShouldEqual, true)
		})
	})
}

func TestIntListContainSubList(t *testing.T) {
	Convey("test IntList.ContainSubList\n", t, func() {
		Convey("1. test not contain sub list\n", func() {
			intList := []int{1, 2, 3}
			ok := IntList(intList).ContainSubList([]int{1, 4})
			So(ok, ShouldEqual, false)
			ok = IntList(intList).ContainSubList([]int{1, 2, 3, 4})
			So(ok, ShouldEqual, false)
			ok = IntList(intList).ContainSubList([]int{5})
			So(ok, ShouldEqual, false)
		})

		Convey("2. test contain sub list\n", func() {
			intList := []int{1, 2, 3}
			ok := IntList(intList).ContainSubList([]int{})
			So(ok, ShouldEqual, true)
			ok = IntList(intList).ContainSubList([]int{1, 2})
			So(ok, ShouldEqual, true)
			ok = IntList(intList).ContainSubList([]int{1, 2, 3})
			So(ok, ShouldEqual, true)
		})
	})
}
