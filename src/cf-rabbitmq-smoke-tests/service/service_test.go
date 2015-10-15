package service_test

import (
	"fmt"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
	"github.com/cloudfoundry-incubator/cf-test-helpers/services/context_setup"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("RabbitMQ Service", func() {
	var timeout = time.Second * 60
	var retryInterval = time.Second * 1
	var appPath = "../assets/cf-rabbitmq-example-app"

	var appName string

	randomName := func() string {
		return uuid.NewRandom().String()
	}

	appUri := func(appName string) string {
		return "https://" + appName + "." + config.AppsDomain
	}

	assertAppIsRunning := func(appName string) {
		pingUri := appUri(appName) + "/ping"
		fmt.Println("Checking that the app is responding at url: ", pingUri)
		Eventually(runner.Curl(pingUri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("OK"))
		fmt.Println("\n")
	}

	BeforeEach(func() {
		appName = randomName()
		Eventually(cf.Cf("push", appName, "-m", "256M", "-p", appPath, "-s", "cflinuxfs2", "-no-start"), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
	})

	AfterEach(func() {
		Eventually(cf.Cf("delete", appName, "-f"), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
	})

	AssertLifeCycleBehavior := func(planName string) {
		It("can create, bind to, write to, read from, unbind, and destroy a service instance using the "+planName+" plan", func() {
			serviceInstanceName := randomName()

			Eventually(cf.Cf("create-service", config.ServiceName, planName, serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
			Eventually(cf.Cf("bind-service", appName, serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
			Eventually(cf.Cf("start", appName), context_setup.ScaledTimeout(5*time.Minute)).Should(Exit(0))
			assertAppIsRunning(appName)

			/*
			    create a queue     (should 201)
			    list the queues    (should 200)
			    subscribe          (should 204)
			    publish            (should 201)
			    subscribe          (should 200)
			 */
			uri := appUri(qppName) + "/queues"
			fmt.Println("Creating a new queue: ", uri)
			Eventually(runner.Curl(ui, "-k", "-X", "POST", "-d", "{name:test-q}"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("SUCCESS"))
			fmt.Println("\n")

			fmt.Println("Listing the queues: ", uri)
			Eventually(runner.Curl(ui, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("test-q\n"))
			fmt.Println("\n")

			uri = appUri(qppName) + "/queue/test-q"
			fmt.Println("Reading from the (empty) queue: ", uri)
			Eventually(runner.Curl(uri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say(""))
			fmt.Println("\n")

			fmt.Println("Publishing to the queue: ", uri)
			Eventually(runner.Curl("-d", "data=test-message", "-X", "PUT", uri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("SUCCESS"))
			fmt.Println("\n")

			fmt.Println("Reading from the (non-empty) queue: ", uri)
			Eventually(runner.Curl(uri, "-k"), context_setup.ScaledTimeout(timeout), retryInterval).Should(Say("test-message"))
			fmt.Println("\n")

			Eventually(cf.Cf("unbind-service", appName, serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
			Eventually(cf.Cf("delete-service", "-f", serviceInstanceName), context_setup.ScaledTimeout(timeout)).Should(Exit(0))
		})
	}

	Context("for each plan", func() {
		for _, planName := range config.PlanNames {
			AssertLifeCycleBehavior(planName)
		}
	})
})
