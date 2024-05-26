package meta

import (
	"fmt"
	"os"
)

const banner = `                 ###########
             ###################
          ######             ######
        ####                    #####
      ####                         ####
     ###                            ####
    ###                              ####
   ####                               ####      Protos      %s
   ####                               ####      PID:        %d
   #######################################      P2P port:   %d
       ###                         ###          Data dir:   %s
       ###############################          Init mode:  %t
       ###############################          Dev mode:   %t
            ///   ///   ///  ////
            ///   ///   ///  ////               https://protos.io
            ///   ///   ///  ////
            ///   ///   ///  ////
            ///   ///   ///  ////
     //     ///   ///   ///  ////    ///
    ////    ///   ///   ///  ////    ///
     ////  ////   ///   ///   ////  ////
      ///////     ///   ///    ////////
                  ///   ///
                  ///   ///
                  ///   ///
                  ///   ///`

// PrintBanner prints the Protos ascii banner
func PrintBanner() {
	pid := os.Getpid()
	fmt.Fprintln(os.Stderr, fmt.Sprintf(
		banner,
		gconfig.Version.String(),
		pid,
		gconfig.P2PPort,
		gconfig.WorkDir,
		gconfig.InitMode,
		gconfig.DevMode))
}
