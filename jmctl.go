package main

import (
    "fmt"
    "os"
    "log"
    "io/ioutil"
    "strconv"
    // "flag"
    "github.com/jawher/mow.cli"
    "github.com/Snowlabs/Jamyxgo"
)

const NAME_ALIAS = "name n na nam -n --name"
const ISIN_ALIAS = "is-input isinput isin i in inp inpu input -in --is-input"
const ISMO_ALIAS = "is-mono ismono ismo ismon ism mono m mo mon -mo --is-mono"
const VOLM_ALIAS = "volume vol vo v -v --volume"
const BALN_ALIAS = "balance bal ba b -b --balance"
const CONS_ALIAS = "connections cons cs c -cs --connections"
const COND_ALIAS = "connected conned connd cond cnd cd -cd --connected"
const MOND_ALIAS = "monitored monned monnd moned mond mnd md -md --monitored"


type Float32Arg float32

func (fla *Float32Arg) String() string {
    return strconv.FormatFloat(float64(*fla), 'f', -1, 32)
}

func (fla *Float32Arg) Set(s string) error {
    val, err := strconv.ParseFloat(s, 32)
    if err != nil {
        return err
    }
    *fla = Float32Arg(float32(val))
    return err
}

func main() {
    var target *jamyxgo.Target
    app := cli.App("jmctl", "Jamyxer control")
    app.Spec = "[-v]"

    verbose := app.BoolOpt("v verbose", false, "Verbose debug mode")

    app.Before = func() {
        target = jamyxgo.NewTarget("localhost", 56065)

        if !*verbose {
            log.SetFlags(0)
            log.SetOutput(ioutil.Discard)
        }

    }

    app.Command("get-all ga geta getall", "get all channels", func(cmd *cli.Cmd) {
        cmd.Spec = "-i|-o"
        isin  := cmd.BoolOpt("i input", false, "input")
        isout := cmd.BoolOpt("o output", false, "output")

        cmd.Action = func() {
            var ports []jamyxgo.Port
            if *isin && !*isout {
                ports = target.GetPorts().Inputs
            } else {
                ports = target.GetPorts().Outputs
            }

            for _, c := range ports {
                fmt.Println(c.Port)
            }
        }
    })
    app.Command("get g ge -g --get", "get a channel's properties", func(cmd *cli.Cmd) {
        var port jamyxgo.Port
        var wait func()

        cmd.Spec = "((-io PORT) | -m) [-VB]"

        name  := cmd.StringArg("PORT", "", "Port name")
        isin  := cmd.BoolOpt("i input", false, "input")
        isout := cmd.BoolOpt("o output", false, "output")
        ismon := cmd.BoolOpt("m monitor", false, "monitor")

        mon_vol := cmd.BoolOpt("V monitor-vol", false, "wait for volume to change")
        mon_bal := cmd.BoolOpt("B monitor-bal", false, "wait for balance to change")

        cmd.Before = func() {
            // if (*isin == *isout) && (*isin == *ismon) { panic("Impossible state!") }
            if !*ismon {
                port = target.GetPort(*isin && !*isout, *name)
            } else {
                port = target.GetMonitorPort()
            }

            if *mon_vol {
                wait = port.ListenVol
            } else if *mon_bal {
                wait = port.ListenBal
            } else {
                wait = func() { /* Do not wait */ }
            }
        }

        construct_cmd := func(name, desc string, attr func() interface{}) {
            cmd.Command(name, desc, func(cmd *cli.Cmd) { cmd.Action = func() { wait(); fmt.Println(attr()) } })
        }

        construct_cmd(NAME_ALIAS, "get a channel's name",    func() interface{} { return port.Port    })
        construct_cmd(ISIN_ALIAS, "is a channel input",      func() interface{} { return port.IsInput })
        construct_cmd(ISMO_ALIAS, "is a channel mono",       func() interface{} { return port.IsMono  })
        construct_cmd(VOLM_ALIAS, "get a channel's volume",  func() interface{} { return port.Vol     })
        construct_cmd(BALN_ALIAS, "get a channel's balance", func() interface{} { return port.Bal     })

        cmd.Command(CONS_ALIAS, "get a channel's connections", func(cmd *cli.Cmd) { cmd.Action = func() {
            wait()
            for _, c := range port.Cons {
                fmt.Println(c)
            }
        } })

        cmd.Command(COND_ALIAS, "is a channel connected to ...", func(cmd *cli.Cmd) {
            name2 := cmd.StringArg("OTHER", "", "Other port name")
            cmd.Action = func() {
                wait()
                for _, e := range port.Cons {
                    if e == *name2 { fmt.Println(true); return }
                }
                fmt.Println(false)
            }
        })

        cmd.Command(MOND_ALIAS, "is a channel monitored", func(cmd *cli.Cmd) { cmd.Action = func() {
            wait()
            m := target.GetMonitorPort()
            fmt.Println((port.Port == m.Port) && (port.IsInput == m.IsInput))
        } })
    })

    app.Command("set se s -s --set", "set a channel's properties", func(cmd *cli.Cmd) {
        var port jamyxgo.Port

        cmd.Spec = "(-io PORT) | -m"

        name  := cmd.StringArg("PORT", "", "Port name")
        isin  := cmd.BoolOpt("i input", false, "input")
        isout := cmd.BoolOpt("o output", false, "output")
        ismon := cmd.BoolOpt("m monitor", false, "monitor")

        cmd.Before = func() {
            // if (*isin == *isout) && (*isin == *ismon) { panic("Impossible state!") }
            if !*ismon {
                port = target.GetPort(*isin && !*isout, *name)
            } else {
                port = target.GetMonitorPort()
            }
        }


        cmd.Command(VOLM_ALIAS, "set a channel's volume", func(cmd *cli.Cmd) {
            val := Float32Arg(0)
            cmd.VarOpt("val v", &val, "value")

            cmd.Action = func() {
                port.SetVol(float32(val))
                fmt.Println(port.Vol)
            }
        })
        cmd.Command(BALN_ALIAS, "set a channel's balance", func(cmd *cli.Cmd) {
            val := Float32Arg(0)
            cmd.VarOpt("val v", &val, "value")

            cmd.Action = func() {
                fmt.Println(float32(val))
                port.SetBal(float32(val))
                fmt.Println(port.Bal)
            }
        })
        cmd.Command(COND_ALIAS, "set a channel connection with ...", func(cmd *cli.Cmd) {
            cmd.Spec = "OTHER -tcd"
            name2 := cmd.StringArg("OTHER", "", "Other port name")
            tog   := cmd.BoolOpt("t toggle", false, "toggle")
            con   := cmd.BoolOpt("c connect", false, "connect")
            dis   := cmd.BoolOpt("d disconnect", false, "disconnect")

            cmd.Action = func() {
                if *tog {
                    port.ToggleConnectionWithChannel(*name2)
                } else if *con {
                    port.ConnectToChannel(*name2)
                } else if *dis {
                    port.DisconnectFromChannel(*name2)
                }

                port.Update() // might be unnecessary
                for _, c := range port.Cons {
                    fmt.Println(c)
                }
            }
        })
        cmd.Command(MOND_ALIAS, "set a channel to be monitored", func(cmd *cli.Cmd) {
            cmd.Action = func() {
                port.SetMonitored()
                fmt.Println(target.GetMonitorPort().Port)
            }
        })

    })


    app.Run(os.Args)

}
