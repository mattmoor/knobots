package gotool

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/injection/client"

	client "github.com/mattmoor/bindings/pkg/github"
	"github.com/mattmoor/knobots/pkg/botinfo"
	"github.com/mattmoor/knobots/pkg/builds"
	"github.com/mattmoor/knobots/pkg/comment"
	"github.com/mattmoor/knobots/pkg/handler"
	"github.com/mattmoor/knobots/pkg/handler/reviewrequest"
	"github.com/mattmoor/knobots/pkg/handler/reviewresult"
	"github.com/mattmoor/knobots/pkg/handler/slack"
)

type gotool struct {
	Client tektonclientset.Interface
}

var _ handler.Interface = (*gotool)(nil)

func New(ctx context.Context) handler.Interface {
	return &gotool{Client: tektonclient.Get(ctx)}
}

func (*gotool) GetType() interface{} {
	return &reviewrequest.Response{}
}

func (gt *gotool) Handle(ctx context.Context, x interface{}) (handler.Response, error) {
	rrr := x.(*reviewrequest.Response)

	// Bot runs as me now, so whack this.
	return nil, nil

	if rrr.Head.GetUser().GetLogin() != "mattmoor" {
		return nil, nil
	}

	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(bytes)

	taskrun := &tektonv1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "gotool-",
			// TODO(mattmoor): Namespace: system.Namespace()?  Needs downward API :(
			Namespace: "default",
		},
		Spec: tektonv1alpha1.TaskRunSpec{
			// 	Source: &tektonv1alpha1.SourceSpec{
			// 		Git: &tektonv1alpha1.GitSourceSpec{
			// 			Url:      rrr.Head.GetRepo().GetCloneURL(),
			// 			Revision: rrr.Head.GetRef(),
			// 		},
			// 	},
			// 	Template: &tektonv1alpha1.TemplateInstantiationSpec{
			// 		Name: "gotool",
			// 		Arguments: []tektonv1alpha1.ArgumentSpec{{
			// 			Name:  "ORGANIZATION",
			// 			Value: rrr.Head.GetUser().GetLogin(),
			// 		}, {
			// 			Name:  "REPOSITORY",
			// 			Value: rrr.Head.GetRepo().GetName(),
			// 		}, {
			// 			Name:  "BRANCH",
			// 			Value: rrr.Head.GetRef(),
			// 		}, {
			// 			Name:  "ASSIGNEE",
			// 			Value: rrr.Head.GetUser().GetLogin(),
			// 		}, {
			// 			// Thread through a token that we hide in the PR body to look up
			// 			// which PR came from this TaskRun.
			// 			Name:  "TOKEN",
			// 			Value: token,
			// 		}},
			// 	},
		},
	}

	var resp handler.Response
	err := builds.Run(gt.Client, taskrun, func(b *tektonv1alpha1.TaskRun) error {
		c := b.Status.GetCondition("Succeeded")
		switch c.Status {
		case "True":
			// Wait for a few seconds after the taskrun's completion to attempt
			// to find the PR it opened.
			time.Sleep(1 * time.Second)

			// TaskRun succeeded
			log.Printf("TaskRun %s succeeded", b.Name)
			// Check for Pull Requests matching the token injected above, and add a
			// comment asking to merge that PR.
			pr, err := findPR(ctx, token, rrr.Head.GetUser().GetLogin(),
				rrr.Head.GetRepo().GetName())
			if err != nil {
				return err
			} else if pr == nil {
				// This will clear any prior review.
				resp = &reviewresult.Payload{
					Name:        botinfo.GetName(),
					Description: `Check for go linting violations.`,
					Owner:       rrr.Owner,
					Repository:  rrr.Repository,
					PullRequest: rrr.PullRequest,
					SHA:         rrr.Head.GetSHA(),
				}
				return nil
			}

			resp = &reviewresult.Payload{
				Name:        botinfo.GetName(),
				Description: `Check for go linting violations.`,
				Owner:       rrr.Owner,
				Repository:  rrr.Repository,
				PullRequest: rrr.PullRequest,
				SHA:         rrr.Head.GetSHA(),
				Body: fmt.Sprintf("Found go linting violations, please merge: %s",
					pr.GetHTMLURL()),
			}

			return nil

		case "False":
			// TaskRun failed
			log.Printf("TaskRun %s failed", b.Name)

			resp = slack.ErrorReport("gotool taskrun failed", map[string]string{
				"pod": fmt.Sprintf("%s/%s", b.Namespace, b.Status.PodName),
			})

			// TODO(mattmoor): Don't bother the nice user...
			// reviewresult.Payload{
			// 	Name:        botinfo.GetName(),
			// 	Description: `Check for go linting violations.`,
			// 	Owner:       rrr.Owner,
			// 	Repository:  rrr.Repository,
			// 	PullRequest: rrr.PullRequest,
			// 	SHA:         rrr.Head.GetSHA(),
			// 	Body:        "TODO: The taskrun failed, include status.",
			// }

			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func findPR(ctx context.Context, token, owner, repo string) (*github.PullRequest, error) {
	ghc, err := client.New(ctx)
	if err != nil {
		return nil, err
	}

	lopt := &github.PullRequestListOptions{}
	for {
		prs, resp, err := ghc.PullRequests.List(ctx, owner, repo, lopt)
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			if comment.HasSignature(token, pr.GetBody()) {
				return pr, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		lopt.Page = resp.NextPage
	}

	return nil, nil
}
