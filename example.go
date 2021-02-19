
package push

import (
	"fmt"
	"math/rand"

	"time"
)

//curl -X DELETE http://10.160.0.172/api/index/metrics -d '{"endpoints":["10.160.0.173"],"metrics":["mt.test.abc.gauge","mt.test.counter.value"]}'
func main() {
	if err := Init("mt.test"); err != nil {
		fmt.Println(err)
		return
	}
	go generateNum()
	select {}
}

//随机数生成,gauge类型指标
func generateNum() {
	for {
		time.Sleep(10 * time.Second)
		Gauge.Set("10.160.0.173", "abc.gauge", rand.Intn(1000000))
		fmt.Println("gauge.vale 推送完成")
	}
}

