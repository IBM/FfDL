/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package trainer

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"
)

func TestQueue(t *testing.T) {
	queue, _ := newTrainingJobQueue(viper.GetString(mongoAddressKey), viper.GetString(mongoDatabaseKey),
		viper.GetString(mongoUsernameKey), viper.GetString(mongoPasswordKey), config.GetMongoCertLocation(),
		fmt.Sprintf("QueueTest%d", time.Now().Unix()), fmt.Sprintf("QueueLockTest%d", time.Now().Unix()))
	t1 := "training job 1"
	t2 := "training job 2"
	t3 := "training job 3"

	queue.Enqueue(t1)
	queue.Enqueue(t2)

	empty, e := queue.Empty()
	assert.Equal(t, false, empty)
	assert.Equal(t, nil, e)

	queue.Enqueue(t3)
	queue.Delete(t2)

	id1, e1 := queue.Dequeue()
	assert.Equal(t, t1, id1)
	assert.Equal(t, nil, e1)

	id3, e3 := queue.Peek()
	assert.Equal(t, t3, id3)
	assert.Equal(t, nil, e3)

	queue.Dequeue()

	id4, e4 := queue.Dequeue()
	assert.Equal(t, "", id4)
	assert.Equal(t, fmt.Errorf("queue is empty"), e4)

	empty, e = queue.Empty()
	assert.Equal(t, true, empty)
	assert.Equal(t, nil, e)

}

// test ordering with 2 threads and locks
func TestLock(t *testing.T) {
	queueCollection := fmt.Sprintf("QueueTest%d", time.Now().Unix())
	lockCollection := fmt.Sprintf("QueueLockTest%d", time.Now().Unix())

	q1, _ := newTrainingJobQueue(viper.GetString(mongoAddressKey), viper.GetString(mongoDatabaseKey),
		viper.GetString(mongoUsernameKey), viper.GetString(mongoPasswordKey), config.GetMongoCertLocation(),
		queueCollection, lockCollection)
	q2, _ := newTrainingJobQueue(viper.GetString(mongoAddressKey), viper.GetString(mongoDatabaseKey),
		viper.GetString(mongoUsernameKey), viper.GetString(mongoPasswordKey), config.GetMongoCertLocation(),
		queueCollection, lockCollection)
	var wg sync.WaitGroup

	// runs represents whether each queue has been locked/unlocked
	runs := []bool{false, false, false, false}

	q1Runner := func(wg *sync.WaitGroup) {
		defer wg.Done()

		err := q1.Lock()
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		runs[0] = true
		assert.Equal(t, []bool{true, false, false, false}, runs)

		time.Sleep(1 * time.Second)

		err = q1.Unlock()
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		runs[1] = true
		assert.Equal(t, []bool{true, true, false, false}, runs)
	}
	q2Runner := func(wg *sync.WaitGroup) {
		defer wg.Done()

		time.Sleep(500 * time.Millisecond)

		err := q2.Lock()
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		runs[2] = true
		assert.Equal(t, []bool{true, true, true, false}, runs)

		err = q2.Unlock()
		if err != nil {
			fmt.Printf("error: %v", err)
		}

		runs[3] = true
		assert.Equal(t, []bool{true, true, true, true}, runs)
	}

	wg.Add(1)
	go q1Runner(&wg)
	wg.Add(1)
	go q2Runner(&wg)

	wg.Wait()
}
