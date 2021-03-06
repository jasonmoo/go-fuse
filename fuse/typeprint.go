package fuse

import (
	"fmt"
	"github.com/jasonmoo/go-fuse/raw"
)

func (me *WithFlags) String() string {
	return fmt.Sprintf("File %s (%s) %s %s",
		me.File, me.Description, raw.FlagString(raw.OpenFlagNames, int64(me.OpenFlags), "O_RDONLY"),
		raw.FlagString(raw.FuseOpenFlagNames, int64(me.FuseFlags), ""))
}

func (a *Attr) String() string {
	return ((*raw.Attr)(a)).String()
}
