package report

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nirmata/kyverno/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	log "sigs.k8s.io/controller-runtime/pkg/log"
)

func HelmCommand() *cobra.Command {
	kubernetesConfig := genericclioptions.NewConfigFlags(true)
	var mode,policy, namespace string
	cmd := &cobra.Command{
		Use:     "helm",
		Short:   "generate report",
		Example: fmt.Sprintf("To create a helm report from background scan:\nkyverno report helm --namespace=defaults \n kyverno report helm"),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			os.Setenv("POLICY-TYPE", "POLICYREPORT")
			restConfig, err := kubernetesConfig.ToRESTConfig()
			if err != nil {
				os.Exit(1)
			}
			const resyncPeriod = 15 * time.Minute
			kubeClient, err := utils.NewKubeClient(restConfig)
			if err != nil {
				log.Log.Error(err, "Failed to create kubernetes client")
				os.Exit(1)
			}
			if mode == "cli" && namespace != "" {
				var wg sync.WaitGroup
				wg.Add(1)
				go backgroundScan(namespace, "Helm",policy,&wg,restConfig)
				wg.Wait()
				return nil
			} else if namespace != "" {
				var wg sync.WaitGroup
				wg.Add(1)
				go configmapScan(namespace, "Helm", &wg, restConfig)
				wg.Wait()
				return nil
			}
			var stopCh <-chan struct{}

			kubeInformer := kubeinformers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod)
			np := kubeInformer.Core().V1().Namespaces()

			go np.Informer().Run(stopCh)

			nSynced := np.Informer().HasSynced

			if !cache.WaitForCacheSync(stopCh, nSynced) {
				log.Log.Error(err, "Failed to create kubernetes client")
				os.Exit(1)
			}
			if mode == "cli" {
				ns, err := np.Lister().List(labels.Everything())
				if err != nil {
					os.Exit(1)
				}
				var wg sync.WaitGroup
				wg.Add(len(ns))
				for _, n := range ns {
					go backgroundScan(n.GetName(), "Helm",policy, &wg, restConfig)
				}
				wg.Wait()
			} else {
				var wg sync.WaitGroup
				wg.Add(1)
				go configmapScan("", "Helm", &wg, restConfig)
				wg.Wait()
				return nil
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "define specific namespace")
	cmd.Flags().StringVarP(&policy, "policy", "p", "", "define specific policy")
	cmd.Flags().StringVarP(&mode, "mode", "m", "cli", "mode")
	return cmd
}
