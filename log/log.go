// Copyright (c) 2014 The AUTHORS
//
// This file is part of trunk.
//
// trunk is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// trunk is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with trunk.  If not, see <http://www.gnu.org/licenses/>.

package log

import (
	stdlog "log"
)

func init() {
	stdlog.SetFlags(0)
}

func Run(msg string) {
	stdlog.Printf("[RUN]  %v\n", msg)
}

func Skip(msg string) {
	stdlog.Printf("[SKIP] %v\n", msg)
}

func Go(msg string) {
	stdlog.Printf("[GO]   %v\n", msg)
}

func Ok(msg string) {
	stdlog.Printf("[OK]   %v\n", msg)
}

func Fail(msg string) {
	stdlog.Printf("[FAIL] %v\n", msg)
}

func Print(v ...interface{}) {
	stdlog.Print(v...)
}

func Printf(format string, v ...interface{}) {
	stdlog.Printf(format, v...)
}

func Println(v ...interface{}) {
	stdlog.Println(v...)
}

func Fatal(v ...interface{}) {
	stdlog.Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	stdlog.Fatalf(format, v...)
}

func Fatalln(v ...interface{}) {
	stdlog.Fatalln(v...)
}
