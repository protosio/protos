package banner

import (
	"fmt"
	"os"

	"github.com/protosio/protos/internal/config"
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
       ###############################
       ###############################          https://protos.io
            ///   ///   ///  ////
            ///   ///   ///  ////
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
func PrintBanner(conf config.Config) {
	pid := os.Getpid()
	fmt.Fprintln(os.Stderr, fmt.Sprintf(
		banner,
		conf.Version.String(),
		pid,
		conf.P2PPort,
		conf.WorkDir))
}
