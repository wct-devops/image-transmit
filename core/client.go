package core

import (
	"container/list"
	"sync"
	"strings"
	"time"
)

type Task interface {
	Run(int) error
	Name() string
	Callback(bool, string)
	StatDown(int64, time.Duration)
	StatUp(int64, time.Duration)
	Status() string
}

// Client describes a syncronization client
type Client struct {
	// a Task list
	taskList *list.List
	// failed list
	failedTaskList *list.List
	invalidTasks   []string
	routineNum     int
	retries        int
	ctx            *TaskContext
	// mutex
	taskListChan       chan int
	failedTaskListChan chan int
	invalidTaskListChan chan int
}

// Newlient creates a syncronization client
func Newlient(routineNum int, retries int, logger *TaskContext) (*Client, error) {
	return &Client{
		taskList:            list.New(),
		failedTaskList:      list.New(),
		routineNum:          routineNum,
		retries:             retries,
		ctx:                 logger,
		taskListChan:        make(chan int, 1),
		failedTaskListChan:  make(chan int, 1),
		invalidTaskListChan: make(chan int, 1),
	}, nil
}

// Run ids main function of a syncronization client
func (c *Client) Run() {
	c.ctx.UpdateInvalidTask(len(c.invalidTasks))
	openRoutinesHandleTaskAndWaitForFinish := func() {
		wg := sync.WaitGroup{}
		for i := 0; i < c.routineNum; i++ {
			wg.Add(1)
			go func(tid int) {
				c.ctx.UpdateCurrentConn(1)
				defer wg.Done()
				defer c.ctx.UpdateCurrentConn(-1)
				for {
					task, empty := c.GetATask()
					c.ctx.UpdateWaitTask(c.taskList.Len())
					// no more tasks need to handle
					if empty {
						break
					}
					if err := task.Run(tid); err != nil {
						errLog := I18n.Sprintf("Task failed with %v", err)
						c.ctx.Error(errLog)
						c.PutAFailedTask(task)
						task.Callback(false, errLog)
					} else {
						task.Callback(true, task.Status())
					}
					c.ctx.UpdateFailedTask(c.failedTaskList.Len())
					if c.ctx.Cancel() {
						c.ctx.Error(I18n.Sprintf("User cancelled..."))
						break
					}
				}
			}(i)
		}
		wg.Wait()
	}

	c.ctx.Info(I18n.Sprintf("Start processing taks, total %v ...", c.taskList.Len()))

	// generate goroutines to handle tasks
	openRoutinesHandleTaskAndWaitForFinish()

	for times := 0; times < c.retries; times++ {
		if c.failedTaskList.Len() != 0 {
			c.taskList.PushBackList(c.failedTaskList)
			c.failedTaskList.Init()
		}

		if c.taskList.Len() != 0 {
			// gzRetries to handle task
			c.ctx.Info(I18n.Sprintf("Start Retry failed tasks"))
			openRoutinesHandleTaskAndWaitForFinish()
		}
	}

	c.ctx.Info(I18n.Sprintf("Task completed, total %v tasks with %v failed", c.ctx.totalTask, c.failedTaskList.Len()))
	if c.failedTaskList.Len() >0 {
		var failListText string
		for e := c.failedTaskList.Front(); e != nil; e = e.Next() {
			t := e.Value.(Task)
			failListText = failListText + t.Name() + "\r\n"
		}
		c.ctx.Info(I18n.Sprintf("Failed tasks:\r\n%s", failListText))
	}
	if len(c.invalidTasks) > 0 {
		c.ctx.Info(I18n.Sprintf("WARNING: there are %v images failed with invalid url(ex:image not exists)", len(c.invalidTasks)))
		c.ctx.Info(I18n.Sprintf("Invalid url list:\r\n%s", strings.Join(c.invalidTasks, "\r\n")))
	}
}

func (c *Client) GenerateOnlineTask(imgSrc string, userSrc string, pswdStr string, imgDst string, userDst string, pswdDst string) error{
	srcURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(imgSrc, "https://"), "http://"))
	imageSourceSrc, err := NewImageSource(c.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), srcURL.GetTag(), userSrc, pswdStr, !strings.HasPrefix(imgSrc,"https") )
	if err != nil {
		c.PutAInvalidTask(imgSrc)
		return c.ctx.Errorf(I18n.Sprintf("Url %s format error: %v, skipped", imgSrc, err))
	}

	dstURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(imgDst, "https://"), "http://"))
	imageSourceDst, err := NewImageDestination(c.ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), dstURL.GetTag(), userDst, pswdDst, !strings.HasPrefix(imgDst,"https") )
	if err != nil {
		c.PutAInvalidTask(imgDst)
		return c.ctx.Errorf(I18n.Sprintf("Url %s format error: %v, skipped", imgDst, err))
	}

	c.PutATask(NewOnlineTask(imageSourceSrc, imageSourceDst, c.ctx))
	c.ctx.Info(I18n.Sprintf("Generated a task for %s to %s", srcURL.GetURL(), dstURL.GetURL()))
	return nil
}

func (c *Client) GenerateOfflineDownTask(url string, username string, password string) error {
	srcURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(url,"https://"),"http://"))
	if err != nil {
		c.PutAInvalidTask(url)
		return c.ctx.Errorf(I18n.Sprintf("Url %s format error: %v, skipped", url, err))
	}
	is, err := NewImageSource(c.ctx.Context, srcURL.GetRegistry(), srcURL.GetRepoWithNamespace(), srcURL.GetTag(),
		username, password, !strings.HasPrefix(url, "https://") )
	if err != nil {
		c.PutAInvalidTask(url)
		return c.ctx.Errorf(I18n.Sprintf("Url %s format error: %v, skipped", url, err))
	}

	c.PutATask(NewOfflineDownTask(c.ctx, url, is))
	c.ctx.Info(I18n.Sprintf("Generated a download task for %s", srcURL.GetURL() ))
	return nil
}

func (c *Client) GenerateOfflineUploadTask(srcUrl string, url string, path string, username string, password string) error {
	var ids *ImageDestination
	if url != "" {
		dstURL, err := NewRepoURL(strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://"))
		ids, err = NewImageDestination(c.ctx.Context, dstURL.GetRegistry(), dstURL.GetRepoWithNamespace(), dstURL.GetTag(), username, password, !strings.HasPrefix(url,"https") )
		if err != nil {
			c.PutAInvalidTask(url)
			return c.ctx.Errorf(I18n.Sprintf("Url %s format error: %v, skipped", url, err))
		}
	}
	c.PutATask(NewOfflineUploadTask(c.ctx, ids, srcUrl, path))
	if url == "" {
		c.ctx.Info(I18n.Sprintf("Generated a upload task for %s", srcUrl ))
	} else {
		c.ctx.Info(I18n.Sprintf("Generated a upload task for %s", url ))
	}
	return nil
}

// GetATask return a Task struct if the task list ids not empty
func (c *Client) GetATask() (Task, bool) {
	c.taskListChan <- 1
	defer func() {
		<-c.taskListChan
	}()

	task := c.taskList.Front()
	if task == nil {
		return nil, true
	}
	c.taskList.Remove(task)

	return task.Value.(Task), false
}

// PutATask puts a Task struct to task list
func (c *Client) PutATask(task Task) {
	c.taskListChan <- 1
	defer func() {
		<-c.taskListChan
	}()

	if c.taskList != nil {
		c.taskList.PushBack(task)
	}
}

// GetAFailedTask gets a failed task from failedTaskList
func (c *Client) GetAFailedTask() (Task, bool) {
	c.failedTaskListChan <- 1
	defer func() {
		<-c.failedTaskListChan
	}()

	failedTask := c.failedTaskList.Front()
	if failedTask == nil {
		return nil, true
	}
	c.failedTaskList.Remove(failedTask)
	return failedTask.Value.(Task), false
}

// PutAFailedTask puts a failed task to failedTaskList
func (c *Client) PutAFailedTask(failedTask Task) {
	c.failedTaskListChan <- 1
	defer func() {
		<-c.failedTaskListChan
	}()
	if c.failedTaskList != nil {
		c.failedTaskList.PushBack(failedTask)
	}
}

func (c *Client) PutAInvalidTask(invalidTask string) {
	c.invalidTaskListChan <- 1
	defer func() {
		<-c.invalidTaskListChan
	}()
	c.invalidTasks = append(c.invalidTasks, invalidTask)
}

func (c *Client) TaskLen() int {
	return c.taskList.Len()
}
