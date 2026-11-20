<template>
  <div class="最底层div">
    <div class="内容div">
      <el-alert type="warning" :closable="false" show-icon
                title="原生插件(.so)在宿主进程内以完整权限运行,等同于服务器权限。只安装你审核过源码、并用本宿主 SDK 编译的插件。"/>
      <div class="工具条">
        <el-button type="primary" icon="Upload" :loading="is上传中" @click="on选择文件('')">
          {{ is上传中 ? (上传进度 < 100 ? '上传中 ' + 上传进度 + '%' : '处理中...') : '上传安装插件' }}
        </el-button>
        <el-button icon="Refresh" @click="on读取列表">刷新</el-button>
        <span class="次要文字" v-if="SDK版本">宿主 SDK 版本: {{ SDK版本 }}</span>
        <input ref="文件框" type="file" accept=".so" hidden @change="on文件已选择">
      </div>

      <el-table v-loading="is加载中" :data="插件列表" border row-key="key" style="width: 100%"
                :header-cell-style="{background:'#FAFAFAFF',color:'#606266'}">
        <!-- 列宽合计约 1100px: 常见的桌面宽度下不出现横向滚动条,否则「插件页面」列会被右侧固定的操作列挡住 -->
        <el-table-column label="插件" min-width="220">
          <template #default="scope">
            <div class="插件名称">{{ scope.row.name }}</div>
            <div class="次要文字" v-if="scope.row.description">{{ scope.row.description }}</div>
            <div class="次要文字" v-if="scope.row.author">作者: {{ scope.row.author }}</div>
            <div class="加载错误" v-if="scope.row.loadError">加载失败: {{ scope.row.loadError }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="key" label="标识" width="120" show-overflow-tooltip/>
        <el-table-column label="版本" width="140">
          <template #default="scope">
            v{{ scope.row.loadedVersion || scope.row.version }}
            <div v-if="scope.row.pendingUpgrade">
              <el-tag type="warning" size="small">升级待重启生效</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status===1?'success':'info'">{{ scope.row.status === 1 ? '已启用' : '已停用' }}</el-tag>
            <div v-if="!scope.row.loaded" style="margin-top: 4px">
              <el-tag type="danger">未加载</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="安装 / 更新时间" width="190">
          <template #default="scope">
            <div>安装 {{ on时间(scope.row.createTime) }}</div>
            <div class="次要文字">更新 {{ on时间(scope.row.updateTime) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="插件页面" min-width="120">
          <template #default="scope">
            <el-button v-for="menu in scope.row.menus || []" :key="menu.Path" link type="primary"
                       @click="on打开插件页面(scope.row, menu)">
              {{ menu.Title || menu.Path }}
            </el-button>
          </template>
        </el-table-column>
        <el-table-column :fixed="is移动端()?false:'right'" label="操作" width="250">
          <template #default="scope">
            <el-button link type="primary" @click="on打开应用配置(scope.row)">应用配置</el-button>
            <el-button link type="primary" @click="on切换状态(scope.row)">
              {{ scope.row.status === 1 ? '停用' : '启用' }}
            </el-button>
            <el-button link type="primary" @click="on选择文件(scope.row.key)">升级</el-button>
            <el-button link type="danger" @click="on卸载(scope.row)">卸载</el-button>
          </template>
        </el-table-column>
        <template v-slot:empty>
          <el-empty description="还没有安装插件"/>
        </template>
      </el-table>
      <div class="次要文字" style="margin-top: 10px">
        安装、停用即时生效;升级和卸载后需要重启系统才彻底生效。开发规范见后端仓库 plugin-dev/README.md
      </div>
    </div>
  </div>

  <!-- v-for 的循环变量必须用英文名: Vue 3.3 的模板编译器不把中文名认作循环变量，
       会把循环里的 v-model 和事件处理缓存成第一项的(改第二个配置项却写到第一个上) -->
  <el-dialog v-model="is应用配置可见" :title="'应用配置 - ' + 当前插件.name" :width="is移动端()?'96%':'760px'"
             destroy-on-close>
    <div v-loading="is应用加载中" class="应用区">
      <el-alert v-if="!当前插件.loaded" type="info" :closable="false" show-icon style="margin-bottom: 10px"
                title="插件当前没有加载,读不到它声明的配置项。这里只能修改各应用的启用状态,已保存的配置会原样保留。"/>
      <el-input v-if="应用列表.length > 6" v-model.trim="应用关键字" placeholder="按应用名称或 AppId 筛选" clearable
                style="margin-bottom: 10px"/>
      <el-empty v-if="!is应用加载中 && 显示的应用.length===0" :description="应用列表.length ? '没有匹配的应用' : '还没有应用'"/>
      <el-collapse v-else accordion>
        <el-collapse-item v-for="app in 显示的应用" :key="app.appId" :name="app.appId">
          <template #title>
            {{ app.appName }} ({{ app.appId }})
            <el-tag size="small" :type="app.已保存状态===1?'success':'info'" style="margin-left: 10px">
              {{ app.已保存状态 === 1 ? '已启用' : '未启用' }}
            </el-tag>
          </template>
          <el-form :label-width="is移动端()?'100px':'150px'" @submit.prevent>
            <el-form-item label="对该应用启用">
              <el-switch v-model="app.status" :active-value="1" :inactive-value="2"/>
            </el-form-item>
            <el-form-item v-for="field in 当前插件.settings || []" :key="field.Key" :label="field.Label || field.Key">
              <el-switch v-if="field.Type==='switch'" v-model="app.settings[field.Key]"
                         active-value="1" inactive-value="0"/>
              <el-select v-else-if="field.Type==='select'" v-model="app.settings[field.Key]" style="width: 100%">
                <el-option v-for="opt in field.Options || []" :key="opt" :label="opt" :value="opt"/>
              </el-select>
              <el-input v-else-if="field.Type==='textarea'" v-model="app.settings[field.Key]" type="textarea" :rows="3"/>
              <el-input v-else v-model="app.settings[field.Key]" :type="field.Type==='number'?'number':'text'"/>
              <div class="次要文字" style="width: 100%" v-if="field.Tip">{{ field.Tip }}</div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="app.保存中" @click="on保存应用配置(app)">保存</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </div>
  </el-dialog>

  <el-dialog v-model="is插件页面可见" width="94%" top="3vh" destroy-on-close>
    <template #header>
      <span>{{ 插件页面标题 }}</span>
      <el-link type="primary" :href="插件页面地址" target="_blank" style="margin-left: 12px">新窗口打开</el-link>
    </template>
    <iframe class="插件页面" :src="插件页面地址"></iframe>
  </el-dialog>
</template>

<script lang="ts" setup>
import {computed, onMounted, ref} from "vue";
import {ElMessage, ElMessageBox} from 'element-plus'
import {GetAppList, GetList, SaveApp, SetStatus, Uninstall, Upload, 取插件页面地址} from "@/api/插件管理api.js";
import {is移动端, 时间_时间戳到时间} from "@/utils/utils";

// 接口失败时 request.js 已经弹出错误提示,这里只处理成功的情况

const is加载中 = ref(false)
const SDK版本 = ref(0)
const 插件列表 = ref<any[]>([])

const on时间 = (秒: number) => 秒 ? 时间_时间戳到时间(秒) : '-'

const on读取列表 = async () => {
  is加载中.value = true
  try {
    const res = await GetList()
    if (res?.code === 10000) {
      SDK版本.value = res.data.sdkVersion
      插件列表.value = res.data.list || []
    }
  } finally {
    is加载中.value = false
  }
}

// ---------- 上传: 安装 / 升级 ----------
const 最大文件字节 = 200 * 1024 * 1024 // 与后端 controller/admin/Plugin.go 的 插件_上传最大字节 一致
const 文件框 = ref<HTMLInputElement>()
const is上传中 = ref(false)
const 上传进度 = ref(0) // 百分比
let 升级key = ''

const on选择文件 = (key: string) => {
  升级key = key
  文件框.value!.value = '' // 清空后,再次选择同一个文件也会触发 change
  文件框.value!.click()
}

const on文件已选择 = async () => {
  const 文件 = 文件框.value?.files?.[0]
  if (!文件) {
    return
  }
  if (!文件.name.endsWith('.so')) {
    ElMessage.error('只支持 .so 插件文件')
    return
  }
  // 后端要收完整个文件才会判断大小,先在这里拦住,免得白传
  if (文件.size <= 0 || 文件.size > 最大文件字节) {
    ElMessage.error('插件文件不能为空,且大小必须在 200MB 以内')
    return
  }
  上传进度.value = 0
  is上传中.value = true
  is加载中.value = true // 上传期间表格显示加载遮罩,避免同时点别的操作; 最后的 on读取列表 会把它关掉
  try {
    const res: any = await Upload(文件, 升级key, (事件: any) => {
      上传进度.value = 事件.total ? Math.floor(事件.loaded / 事件.total * 100) : 0
    })
    const 状态码 = res?.response?.status
    if (res?.code === 10000) {
      ElMessage.success(res.msg)
    } else if (状态码 && 状态码 !== 404 && 状态码 !== 500) {
      // request.js 只对 404/500 弹出提示。其它状态码(常见的是反向代理限制了请求体大小)在这里提示,否则页面没有任何反应
      ElMessage.error(状态码 === 413
          ? '上传失败: 文件超过了反向代理允许的大小(HTTP 413),请调大 nginx 的 client_max_body_size 后重试'
          : '上传失败: 服务器返回 HTTP ' + 状态码)
    }
  } finally {
    is上传中.value = false
    await on读取列表()
  }
}

// ---------- 启用 / 停用 / 卸载 ----------
const on切换状态 = async (插件: any) => {
  const res = await SetStatus({key: 插件.key, status: 插件.status === 1 ? 2 : 1})
  if (res?.code === 10000) {
    ElMessage.success(res.msg)
    await on读取列表()
  }
}

const on卸载 = async (插件: any) => {
  try {
    await ElMessageBox.confirm(
        '应用配置会被删除;插件自建的数据表(plg_' + 插件.key + '_*)和数据目录保留。',
        '确定卸载插件「' + 插件.name + '」?',
        {confirmButtonText: '卸载', cancelButtonText: '取消', type: 'warning'})
  } catch {
    return
  }
  const res = await Uninstall({key: 插件.key})
  if (res?.code === 10000) {
    ElMessage.success(res.msg)
    await on读取列表()
  }
}

// ---------- 按应用的启用状态与配置 ----------
const is应用配置可见 = ref(false)
const is应用加载中 = ref(false)
const 当前插件 = ref<any>({})
const 应用列表 = ref<any[]>([])
const 应用关键字 = ref('')
const 显示的应用 = computed(() => {
  const 关键字 = 应用关键字.value.toLowerCase()
  if (!关键字) {
    return 应用列表.value
  }
  return 应用列表.value.filter((项: any) =>
      String(项.appId).includes(关键字) || String(项.appName).toLowerCase().includes(关键字))
})

let 应用请求序号 = 0

const on打开应用配置 = async (插件: any) => {
  const 序号 = ++应用请求序号
  当前插件.value = 插件
  应用列表.value = []
  应用关键字.value = ''
  is应用配置可见.value = true
  is应用加载中.value = true
  let res: any
  try {
    res = await GetAppList({key: 插件.key})
  } finally {
    // 等待期间又打开了另一个插件的配置: 这次的结果已经过期,不能覆盖后打开的那个
    if (序号 === 应用请求序号) {
      is应用加载中.value = false
    }
  }
  if (序号 !== 应用请求序号 || res?.code !== 10000) {
    return
  }
  应用列表.value = (res.data.list || []).map((项: any) => {
    // 配置值统一是字符串。原配置 留着: 插件未加载时拿不到字段声明,保存时要把已有配置原样带回去,否则会被清空
    const 原配置: Record<string, string> = {}
    for (const 键 of Object.keys(项.settings || {})) {
      原配置[键] = String(项.settings[键])
    }
    const 配置: Record<string, string> = {}
    for (const 字段 of 插件.settings || []) {
      let 值 = 原配置[字段.Key] ?? ''
      if (字段.Type === 'switch') {
        值 = (值 === '1' || 值 === 'true') ? '1' : '0'
      }
      配置[字段.Key] = 值
    }
    return {appId: 项.appId, appName: 项.appName, status: 项.status, 已保存状态: 项.status, 原配置, settings: 配置, 保存中: false}
  })
}

const on保存应用配置 = async (应用: any) => {
  const 配置: Record<string, string> = {...应用.原配置}
  for (const 键 of Object.keys(应用.settings)) {
    配置[键] = String(应用.settings[键] ?? '')
  }
  应用.保存中 = true
  let res: any
  try {
    res = await SaveApp({key: 当前插件.value.key, appId: 应用.appId, status: 应用.status, settings: 配置})
  } finally {
    应用.保存中 = false
  }
  if (res?.code === 10000) {
    应用.已保存状态 = 应用.status
    应用.原配置 = 配置
    ElMessage.success(res.msg)
  }
}

// ---------- 插件自带的管理页面 ----------
const is插件页面可见 = ref(false)
const 插件页面标题 = ref('')
const 插件页面地址 = ref('')

const on打开插件页面 = (插件: any, 菜单: any) => {
  插件页面标题.value = 插件.name + ' - ' + (菜单.Title || 菜单.Path)
  插件页面地址.value = 取插件页面地址(插件.key, 菜单.Path)
  is插件页面可见.value = true
}

onMounted(on读取列表)
</script>

<style scoped lang="scss">
.最底层div {
  min-height: calc(100vh - 200px);
  padding: 12px 16px;
  margin: 0 2px 10px;
  background: #f0f2f5;
}

.内容div {
  min-height: 20%;
  padding: 12px 16px;
  margin: 0 2px 10px;
  background: #ffffff;
}

.工具条 {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin: 12px 0;

  .el-button {
    margin-left: 0;
  }
}

.插件名称 {
  font-weight: 600;
}

// 应用很多时在对话框内滚动,不把整个对话框撑到屏幕外
.应用区 {
  min-height: 120px;
  max-height: 64vh;
  overflow-y: auto;
}

.次要文字 {
  color: #909399;
  font-size: 12px;
  line-height: 1.6;
}

.加载错误 {
  color: #f56c6c;
  font-size: 12px;
  line-height: 1.6;
  word-break: break-all;
}

.插件页面 {
  display: block;
  width: 100%;
  height: 76vh;
  border: 1px solid #ebeef5;
  border-radius: 4px;
}
</style>
