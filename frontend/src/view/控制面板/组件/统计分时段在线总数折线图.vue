<template>
  <div v-loading="is加载中"  element-loading-text="数据努力统计中..." class="dashboard-line-box" style="width: 100%;min-height: 360px;position:relative">
    <!--这里是echarts绘图区域-->
    <div
        ref="echart"
        class="dashboard-line"
        style="width: 100%;min-height: 200px;z-index:99;position:absolute">
    </div>
  </div>
</template>
<script setup>
import * as echarts from 'echarts'
import {nextTick, onMounted, onUnmounted, ref, shallowRef} from 'vue'
import {is移动端} from "@/utils/utils";
import {Get统计分时段在线总数} from "@/api/分析页Api.js";

const Props = defineProps({
  AppId: {
    type: Number,
    default: 0
  }
})
const is加载中 = ref(false)
const chart = shallowRef(null)
const echart = ref(null)
const initChart = () => {
  chart.value = echarts.init(echart.value /* 'macarons' */)
  setOptions(['0时', '1时', '2时', '3时', '4时', '5时', '6时'],[
    {
      name: '分时在线统计',
      type: 'line',
      data: [9920, 332, 341, 354, 390, 220, 9450]
    }
  ])
}

const setOptions = (横轴,data) => {
  let 图数据 = {
    title: {
      text: '分时在线统计'
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        // Use axis to trigger tooltip
        type: 'shadow' // 'shadow' as default; can also be 'line' or 'shadow'
      },
    },
    legend: {
      data: data.map((曲线) => 曲线.name)
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    toolbox: {
      feature: {
        saveAsImage: {}
      }
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: 横轴
    },
    yAxis: {
      type: 'value'
    },
    series: data
  }
  图数据.title = is移动端() ? "" : 图数据.title
  chart.value.setOption(图数据)
}
const on读取图表数据 = async () => {
  is加载中.value=true
  let 返回;
  返回 = await Get统计分时段在线总数({AppId:Props.AppId})
  is加载中.value=false
  console.log(返回)
  if (返回.code === 10000 && Array.isArray(返回.data) && 返回.data.length > 0) {
    // 后端返回的最后一项是横轴(各时段)，前面每一项是一条曲线(今日/昨日/前日)
    const 横轴 = 返回.data[返回.data.length - 1].data
    setOptions(横轴, 返回.data.slice(0, -1))
  }
}
onMounted(async () => {
  await nextTick()
  initChart()
  await on读取图表数据()
})

onUnmounted(() => {
  if (!chart.value) {
    return
  }
  chart.value.dispose()
  chart.value = null
})
window.onresize = function () {
  if (chart.value){
    chart.value.resize();
  }
}
</script>
<style lang="scss" scoped>
.dashboard-line-box {
  .dashboard-line {
    background-color: #fff;
    height: 360px;
    width: 100%;
  }

  .dashboard-line-title {
    font-weight: 600;
    margin-bottom: 12px;
  }
}
</style>
