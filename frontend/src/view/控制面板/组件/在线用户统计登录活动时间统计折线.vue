<template>
  <div v-loading="is加载中"  element-loading-text="数据努力统计中..." class="dashboard-line-box" style="width: 100%;min-height: 360px;position:relative">
    <!--这里是echarts绘图区域-->
    <div
        ref="echart"
        class="dashboard-line"
        style="width: 100%;min-height: 200px;z-index:99;position:absolute">
    </div>
    <!--这里是放置按钮让其显示在最前面-->
    <div style="padding-left: 120px; z-index:999;float:left;position:absolute">
      <el-radio-group v-model="图表时间单位" size="small" @change="on读取图表数据">
        <el-radio-button :value="1">单位(日)</el-radio-button>
        <el-radio-button :value="2">(月)</el-radio-button>
      </el-radio-group>
    </div>
  </div>
</template>
<script setup>
import * as echarts from 'echarts'
import {nextTick, onMounted, onUnmounted, ref, shallowRef} from 'vue'
import {is移动端} from "@/utils/utils";
import {get图表在线用户登录活动时间} from "@/api/分析页Api.js";

const is加载中 = ref(false)
const 图表时间单位 = ref(1)
const chart = shallowRef(null)
const echart = ref(null)
const initChart = () => {
  chart.value = echarts.init(echart.value /* 'macarons' */)
  setOptions(1,[])
}

// 后端返回的最后一个点是今天(或本月)，往前依次是前一天(或前一个月)
const 生成横轴 = (单位, 点数) => {
  const 横轴 = []
  for (let i = 点数 - 1; i >= 0; i--) {
    const 日期 = new Date()
    if (单位 === 2) {
      日期.setDate(1)
      日期.setMonth(日期.getMonth() - i)
      横轴.push(日期.getFullYear() + '-' + String(日期.getMonth() + 1).padStart(2, '0'))
    } else {
      日期.setDate(日期.getDate() - i)
      横轴.push((日期.getMonth() + 1) + '/' + 日期.getDate())
    }
  }
  return 横轴
}

const setOptions = (单位,data) => {

  let 图数据 = {
    title: {
      text: '用户活跃统计'
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
      data: 生成横轴(单位, data.length ? data[0].data.length : 0)
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
  返回 = await get图表在线用户登录活动时间({Type:图表时间单位.value})
  is加载中.value=false
  console.log(返回)
  if (返回.code === 10000 && Array.isArray(返回.data)) {
    setOptions(图表时间单位.value,返回.data)
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
