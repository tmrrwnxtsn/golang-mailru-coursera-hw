package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})
	for _, theJob := range jobs {
		wg.Add(1)
		out := make(chan interface{})
		go func(job job, in, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			job(in, out)
		}(theJob, in, out, wg)
		in = out
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for data := range in {
		wg.Add(1)
		go func(data int, out chan interface{}, wg *sync.WaitGroup, mu *sync.Mutex) {
			defer wg.Done()
			strData := strconv.Itoa(data)

			mu.Lock()
			md5data := DataSignerMd5(strData)
			mu.Unlock()

			crc32Chan := make(chan string)
			go func(data string, out chan string) {
				out <- DataSignerCrc32(strData)
			}(strData, crc32Chan)

			crc32md5Chan := make(chan string)
			go func(data string, out chan string) {
				out <- DataSignerCrc32(md5data)
			}(md5data, crc32md5Chan)

			out <- <-crc32Chan + "~" + <-crc32md5Chan
		}(data.(int), out, wg, mu)
	}

	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for data := range in {
		wg.Add(1)
		go func(data string, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()

			mhData := make([]string, 6)
			wgIn := &sync.WaitGroup{}
			muIn := &sync.Mutex{}

			for th := 0; th < 6; th++ {
				wgIn.Add(1)
				thData := strconv.Itoa(th) + data
				go func(data string, th int, mhData []string, wg *sync.WaitGroup, mu *sync.Mutex) {
					defer wgIn.Done()

					res := DataSignerCrc32(thData)

					muIn.Lock()
					mhData[th] = res
					muIn.Unlock()
				}(thData, th, mhData, wgIn, muIn)
			}

			wgIn.Wait()
			out <- strings.Join(mhData, "")
		}(data.(string), out, wg)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	results := make([]string, 0)
	for data := range in {
		results = append(results, data.(string))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}
