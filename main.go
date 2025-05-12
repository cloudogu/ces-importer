package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/ces-importer/api/exporter"
	"github.com/cloudogu/ces-importer/api/importer"
	"github.com/cloudogu/ces-importer/configuration"
	"github.com/cloudogu/ces-importer/cron"
	"github.com/cloudogu/ces-importer/logging"
	"github.com/cloudogu/ces-importer/mail"
	"github.com/cloudogu/ces-importer/sync"
	"github.com/cloudogu/ces-importer/systeminfo"
	ecoSystemV2 "github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"io"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"log/slog"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	"syscall"
	"time"
)

const hostProtocolScheme = "https://"
const keyFQDN = "fqdn"
const migrationJobLabelSelector = "app.kubernetes.io/instance=ces-exporter" // TODO: Replace with real selector after job actually exists
const jobLogFile = "/home/ces-importer/migration-log/job.log"

var initializeLogging = logging.Initialize
var copyLogsToContainer = func(ctx context.Context, mlc *mainLoopContext) (string, error) {
	return mlc.copyLogsToContainer(ctx)
}

func main() {
	ctx := context.Background()

	cfg, err := configuration.ReadConfigFromEnv()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %w", err))
	}

	err = initializeLogging(cfg)
	if err != nil {
		panic(err)
	}

	logUsedConfig(cfg)

	httpClient := &http.Client{}
	exportApiCli := exporter.NewClient(cfg.ExporterApiKey, httpClient)

	k8sRestConfig, err := ctrl.GetConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read kube config: %w", err))
	}

	kubernetesClient, err := kubernetes.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create kube-client: %w", err))
	}
	pvcClient := kubernetesClient.CoreV1().PersistentVolumeClaims(cfg.ImporterNamespace)

	doguCli, err := ecoSystemV2.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create dogu client: %w", err))
	}

	k8sClient, err := kubernetes.NewForConfig(k8sRestConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create k8s client: %w", err))
	}

	globalConfig := repository.NewGlobalConfigRepository(k8sClient.CoreV1().ConfigMaps(cfg.ImporterNamespace))

	doguClient := doguCli.Dogus(cfg.ImporterNamespace)
	doguStartStopper := importer.NewDoguDeploymentClient(doguClient)

	syncer := sync.NewRsyncSyncer(cfg.ExporterHost, cfg.ExporterSSHUser, cfg.ImporterPrivateSSHKeyPath)

	provider, err := systeminfo.NewSystemInfoProvider(cfg.ImporterNamespace)
	if err != nil {
		panic(fmt.Errorf("failed to create system info provider: %w", err))
	}
	validator, err := systeminfo.NewValidator(cfg, provider, doguClient, pvcClient)
	if err != nil {
		panic(fmt.Errorf("failed to create validator: %w", err))
	}

	mainLoopCtx := mainLoopContext{
		config:           cfg,
		exportApiCli:     exportApiCli,
		doguStart:        doguStartStopper,
		doguStop:         doguStartStopper,
		syncer:           syncer,
		sysInfoValidator: validator,
		globalConfig:     globalConfig,
		mailSender:       smtp.SendMail,
		remove:           os.Remove,
		create: func(name string) (file, error) {
			return os.Create(name)
		},
		initLogging: logging.Initialize,
		pods:        k8sClient.CoreV1().Pods(cfg.ImporterNamespace),
	}
	mainLoop := mainLoopCtx.createMainLoop()

	cronLooper, err := cron.New(ctx, cfg.MigrationRegularCron, mainLoop)
	if err != nil {
		panic(fmt.Errorf("failed to create cron looper for expression %q: %w", cfg.MigrationRegularCron, err))
	}

	// Wait for interrupt signals to gracefully shut down the server with a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		<-quit
		slog.Info("Shutdown Server ...")

		<-ctx.Done()
		cronLooper.Stop()
		slog.Warn("shutdown reached: exiting")
	}()

	slog.Info("Starting main loop")
	cronLooper.Run()
}

type systemInfoValidator interface {
	ValidateSystemInfo(ctx context.Context) error
}

type file interface {
	io.Writer
	WriteString(s string) (n int, err error)
	Close() error
}

type osRemove func(name string) error
type osCreate func(name string) (file, error)
type osReadFile func(name string) ([]byte, error)
type loggingInitializer func(conf configuration.Configuration) error
type podInterface interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
	GetLogs(name string, opts *v1.PodLogOptions) *restclient.Request
}
type globalConfig interface {
	Get(ctx context.Context) (config.GlobalConfig, error)
}

var streamLogs = func(ctx context.Context, req *restclient.Request) (io.ReadCloser, error) {
	return req.Stream(ctx)
}

type mainLoopContext struct {
	config           configuration.Configuration
	exportApiCli     exporterApiClient
	doguStart        doguStarter
	doguStop         doguStopper
	syncer           doguVolumeSyncer
	sysInfoValidator systemInfoValidator
	globalConfig     globalConfig
	pods             podInterface
	mailSender       mail.SenderService
	remove           osRemove
	create           osCreate
	initLogging      loggingInitializer
	readFile         osReadFile
}

func (mlc *mainLoopContext) createMainLoop() func(ctx context.Context) (int, error) {
	return func(ctx context.Context) (int, error) {
		startTime := time.Now()
		err := mlc.initLogging(mlc.config)
		if err != nil {
			return 0, fmt.Errorf("failed to reset logger: %w", err)
		}
		sender := mail.CreateSender(mlc.config.MailConfig, mlc.mailSender, mail.OsReadFile(mlc.readFile))

		retVal, err := func() (int, error) {
			isExporterSyncReady, err := isApiExportReady(ctx, mlc.config.ExporterHost, mlc.exportApiCli)
			if err != nil {
				// This error is recoverable except for misconfiguration, which may be detected by analyzing the logs.
				slog.Error(fmt.Sprintf("Error while checking export sync readiness: %s", err.Error()))
				slog.Info("Waiting for the next run...")
				return 0, nil
			}

			if !isExporterSyncReady {
				// This condition is recoverable, but it is still unclear when the ready status will be triggered
				slog.Info("Exporter does not seem to be ready. Waiting for the next run...")
				return 0, nil // continue to the next main loop iteration
			}

			systemInfo, err := fetchExporterSystemInfo(ctx, mlc.config.ExporterHost, mlc.exportApiCli)
			if err != nil {
				// this error is recoverable, the exporter system API might be down, or the API server errs
				slog.Error(fmt.Sprintf("Failed to fetch the system info from the exporter: %s", err.Error()))
				slog.Info("Waiting for the next run...")
				return 0, nil
			}

			err = mlc.sysInfoValidator.ValidateSystemInfo(ctx)
			if err != nil {
				slog.Log(ctx, slog.LevelError, fmt.Sprintf("Failed to validate importer system info: %s", err.Error()))
				// TODO should this break the main loop or not?
				slog.Log(ctx, slog.LevelInfo, "Waiting for the next run...")
				return 0, nil
			}

			err = deactivateImporterDogus(ctx, systemInfo, mlc.doguStop)
			if err != nil {
				return 1, err
			}

			err = syncDogus(ctx, systemInfo, mlc.exportApiCli, mlc.syncer)
			if err != nil {
				return 2, err
			}

			err = activateImporterDogus(ctx, systemInfo, mlc.doguStart)
			if err != nil {
				return 3, err
			}

			slog.Info("Sync successful")

			return 0, nil
		}()

		if err != nil {
			return retVal, err
		}

		logFilesToAppend := []string{logging.AppLogFile}

		logFile, err := copyLogsToContainer(ctx, mlc)
		if err != nil {
			return 0, fmt.Errorf("failed to collect import job logs: %w", err)
		} else if logFile != "" {
			logFilesToAppend = append(logFilesToAppend, logFile)
		}

		global, err := mlc.globalConfig.Get(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get global config: %w", err)
		}

		fqdn, _ := global.Get(keyFQDN)

		success := err != nil
		err = sender.SendMigrationResult(
			success,
			logFilesToAppend,
			hostProtocolScheme+mlc.config.ExporterHost,
			hostProtocolScheme+fqdn.String(),
			startTime,
			time.Now(),
			false, // TODO: How to determine if final or not?
		)

		if err != nil {
			return 0, fmt.Errorf("failed to send mail: %w", err)
		}

		return retVal, err
	}
}

func (mlc *mainLoopContext) copyLogsToContainer(ctx context.Context) (string, error) {
	err := mlc.remove(jobLogFile)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to clear old job log file: %w", err)
	}

	logFile, err := mlc.create(jobLogFile)
	if err != nil {
		return "", fmt.Errorf("failed to create log file: %w", err)
	}
	defer func() {
		if logFile != nil {
			_ = logFile.Close()
		}
	}()

	pods, err := mlc.pods.List(ctx, metav1.ListOptions{
		LabelSelector: migrationJobLabelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods with matching label: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", nil
	}

	for _, pod := range pods.Items {
		err = func() error {
			req := mlc.pods.GetLogs(pod.Name, &v1.PodLogOptions{})

			podLogs, err := streamLogs(ctx, req)
			if err != nil {
				return fmt.Errorf("error opening log stream for pod %s: %w", pod.Name, err)
			}
			defer func() {
				if podLogs != nil {
					_ = podLogs.Close()
				}
			}()

			_, err = logFile.WriteString(fmt.Sprintf("=== Logs for Pod: %s ===\n", pod.Name))
			if err != nil {
				return fmt.Errorf("failed to write to log file: %w", err)
			}

			_, err = io.Copy(logFile, podLogs)
			if err != nil {
				return fmt.Errorf("failed to copy log for pod %s: %w", pod.Name, err)
			}

			_, err = logFile.WriteString("\n\n")
			if err != nil {
				return fmt.Errorf("failed to write to log file: %w", err)
			}

			return nil
		}()
		if err != nil {
			return "", fmt.Errorf("failed to get log for pod %s: %w", pod.Name, err)
		}
	}

	return jobLogFile, nil
}

func deactivateImporterDogus(ctx context.Context, systemInfo *exporter.SystemInfo, doguStop doguStopper) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Info("Deactivating dogu ", "doguName", dogu.Name)

		err := doguStop.StopDogu(ctx, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to deactivate dogu %s in the importer: %w", dogu.Name, err)
		}
	}

	return nil
}

func activateImporterDogus(ctx context.Context, systemInfo *exporter.SystemInfo, doguStart doguStarter) error {
	for _, dogu := range systemInfo.Dogus {
		slog.Info("Activating dogu", "doguName", dogu.Name)

		err := doguStart.StartDogu(ctx, dogu)
		if err != nil {
			// this error does not seem recoverable because the dogu must be down to avoid copy data problems
			return fmt.Errorf("failed to activate dogu %s in the importer: %w", dogu.Name, err)
		}
	}

	return nil
}

func syncDogus(ctx context.Context, systemInfo *exporter.SystemInfo, _ exporterApiClient, syncer doguVolumeSyncer) error {
	// FIXME: #4: actually implement the core functionality in a proper way. This is part of an upcoming feature

	for _, dogu := range systemInfo.Dogus {
		slog.Info("Starting sync for dogu ", "doguName", dogu.Name)

		exporterSource, importerDestination, exporterPort := func(dogu exporter.Dogu) (string, string, string) {
			return "call your your exporterApiClient for data here", "and here", "and here"
		}(dogu)

		if err := syncer.SyncDogu(ctx, exporterPort, exporterSource, importerDestination); err != nil {
			// TODO: should we continue syncing other dogus on a best-effort basis?
			return fmt.Errorf("failed to sync source %s to destination %s: %w", exporterSource, importerDestination, err)
		}
		slog.Info("Syncing for dogu successful", "doguName", dogu.Name)
	}

	return nil

}

func isApiExportReady(ctx context.Context, hostname string, apiCli exporterApiClient) (isActive bool, err error) {
	exporterUrl := hostProtocolScheme + hostname + exporter.EndpointExportMode

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return false, fmt.Errorf("failed to check whether exporter is export ready: %w", err)
	}

	var exportMode exporter.ExportMode
	err = json.Unmarshal(result, &exportMode)
	if err != nil {
		return false, fmt.Errorf("failed to parse export mode response: %q: %w", result, err)
	}

	return exportMode.IsActive, nil
}

func fetchExporterSystemInfo(ctx context.Context, hostname string, apiCli exporterApiClient) (*exporter.SystemInfo, error) {
	exporterUrl := hostProtocolScheme + hostname + exporter.EndpointSystemInfo

	result, err := apiCli.DoGetRequest(ctx, exporterUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch exporter system info: %w", err)
	}

	var systemInfo exporter.SystemInfo
	err = json.Unmarshal(result, &systemInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse system info response: %q: %w", result, err)
	}

	return &systemInfo, nil
}

func logUsedConfig(config configuration.Configuration) {
	slog.Info("                     ./////,                    ")
	slog.Info("                 ./////==//////,                ")
	slog.Info("                ////.  ___   ////.              ")
	slog.Info("         ,OO,. ////  ,////A,  */// ,OO,.        ")
	slog.Info("    ,/////////////*  */////*  *////////////A    ")
	slog.Info("   ////'        `VA.   '|'   .///'       '///*  ")
	slog.Info("  *///  .*///*,         |         .*//*,   ///* ")
	slog.Info("  (///  (//////)**--_./////_----*//////)   ///) ")
	slog.Info("   V///   '°°°°      (/////)      °°°°'   ////  ")
	slog.Info("    V/////(////////o. '°°°' ./////////(///(/'   ")
	slog.Info("       'V/(/////////////////////////////V'      ")

	// Set api key to empty value to prevent logging secrets
	config.ExporterApiKey = "<removed for log output>"

	slog.Info("ces-importer started using this configuration:",
		"config", fmt.Sprintf("%#v", config),
	)
}
