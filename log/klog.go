package log

import (
	goflag "flag"
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"sync"
)

var (
	KlogScope     = RegisterScope("klog", "", 0)
	configureKlog = sync.Once{}
)

func EnableKlogWithCobra() {
	gf := klogVerboseFlag()
	pflag.CommandLine.AddFlag(pflag.PFlagFromGoFlag(
		&goflag.Flag{
			Name:     "vklog",
			Value:    gf.Value,
			DefValue: gf.DefValue,
			Usage:    gf.Usage + ". Like -v flag. ex: --vklog=9",
		}))
}

func EnableKlogWithGoFlag() {
	gf := klogVerboseFlag()
	goflag.CommandLine.Var(gf.Value, "vklog", gf.Usage+". Like -v flag. ex: --vklog=9")
}

func klogVerbose() bool {
	gf := klogVerboseFlag()
	return gf.Value.String() != "0"
}

var (
	klogFlagSet     = &goflag.FlagSet{}
	klogFlagSetOnce = sync.Once{}
)

func klogVerboseFlag() *goflag.Flag {
	klogFlagSetOnce.Do(func() {
		klog.InitFlags(klogFlagSet)
	})
	// --v= flag of klog.
	return klogFlagSet.Lookup("v")
}

func EnableKlogWithVerbosity(v int) {
	_ = klogFlagSet.Set("v", fmt.Sprint(v))
}
