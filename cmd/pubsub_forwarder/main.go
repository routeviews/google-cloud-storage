package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/golang/glog"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

type server struct {
	loc     string
	queue   string
	project string
	tc      *cloudtasks.Client
}

func (s *server) forwardPubsubMessage(w http.ResponseWriter, r *http.Request) {
	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", s.project, s.loc, s.queue)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Error(err)
		return
	}
	glog.Infof("Received message: %s", string(body))

	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_AppEngineHttpRequest{
				AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
					HttpMethod:  taskspb.HttpMethod_POST,
					RelativeUri: "/",
					Body:        body,
				},
			},
		},
	}

	createdTask, err := s.tc.CreateTask(r.Context(), req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		glog.Error(err)
		return
	}
	glog.Infof("Created task: %+v", createdTask)
}

func main() {
	ctx := context.Background()
	flag.Parse()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		glog.Infof("Defaulting to port %s", port)
	}
	project := os.Getenv("PROJECT")
	if project == "" {
		glog.Fatalf("project not specified")
	}
	loc := os.Getenv("CLOUD_TASK_LOCATION")
	if loc == "" {
		glog.Fatalf("location not specified")
	}
	queue := os.Getenv("CLOUD_TASK_QUEUE")
	if loc == "" {
		glog.Fatalf("queue not specified")
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		glog.Fatalf("cloudtasks.NewClient: %v", err)
	}
	defer client.Close()

	srvr := &server{
		project: project,
		loc:     loc,
		queue:   queue,
		tc:      client,
	}

	http.HandleFunc("/", srvr.forwardPubsubMessage)
	glog.Infof("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		glog.Fatal(err)
	}
}
