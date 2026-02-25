import plugin from "../../lib/plugins/plugin.js"
import fs from "node:fs"
import path from "node:path"
import lodash from "lodash"
import moment from "moment"

const PLUGIN_NAME = "ChatCorpus"
const DATA_DIR = path.join(process.cwd(), "data", PLUGIN_NAME)
const LOG_FILE = path.join(DATA_DIR, "chat_history.jsonl")
const EXPORT_DIR = path.join(process.cwd(), "temp", PLUGIN_NAME)

// 去敏
const REGEX_PHONE = /(?<!\d)(1[3-9]\d{9})(?!\d)/g
const REGEX_EMAIL = /([a-zA-Z0-9._-]+@[a-zA-Z0-9._-]+\.[a-zA-Z0-9._-]+)/g
const REGEX_ID_CARD = /(?<!\d)(\d{6})(19|20)(\d{2})(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])(\d{3}[\dXx])(?!\d)/g

// N-gram 配置
const N_GRAM_SIZE = 2
const TIME_WINDOW = 120

export class ChatCorpusCollector extends plugin {
  constructor() {
    super({
      name: "语料收集与处理",
      dsc: "收集群聊记录并处理为 Pretraining 语料",
      event: "message",
      priority: 5000,
      rule: [
        { reg: "^#生成(预训练)?语料$", fnc: "generateCorpus", permission: "master" },
        { reg: "", fnc: "collectMessage", log: false }
      ]
    })
  }

  async init() {
    const dirs = [DATA_DIR, EXPORT_DIR]
    for (const dir of dirs) {
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true })
      }
    }
  }

  /**
   * 实时收集消息
   */
  async collectMessage(e) {
    // 只记录群聊，且忽略机器人自己发的消息
    if (!e.isGroup || e.user_id === e.self_id) return false

    // 获取纯文本内容
    let rawText = e.msg || ""
    
    // 如果消息包含图片等非文本元素，尝试提取其中的文本部分
    if (Array.isArray(e.message)) {
        rawText = e.message
            .filter(item => item.type === 'text')
            .map(item => item.text)
            .join(' ')
            .trim()
    }

    if (!rawText || rawText.length < 2) return false // 太短的消息忽略

    // 1. PII Masking (个人信息去敏)
    const maskedText = this.piiMasking(rawText)

    // 2. 构建数据对象
    const record = {
      group_id: e.group_id,
      user_id: e.user_id,
      time: e.time || Math.floor(Date.now() / 1000),
      msg_id: e.message_id,
      // 尝试获取回复引用的ID (不同适配器字段可能不同，这里兼容处理)
      reply_id: e.source?.message_id || null, 
      content: maskedText
    }

    // 3. 追加写入文件 (JSONL格式)
    try {
      fs.appendFileSync(LOG_FILE, JSON.stringify(record) + "\n")
    } catch (err) {
      logger.error(`[${PLUGIN_NAME}] 写入日志失败:`, err)
    }

    return false // 返回false以允许消息继续向下传递给其他插件
  }

  /**
   * 生成语料主函数
   */
  async generateCorpus(e) {
    if (!fs.existsSync(LOG_FILE)) {
      return await e.reply("暂无聊天记录数据", true)
    }

    await e.reply("开始处理语料，这可能需要一些时间...", true)

    try {
      const rawData = fs.readFileSync(LOG_FILE, "utf-8")
        .split("\n")
        .filter(line => line.trim())
        .map(line => {
            try { return JSON.parse(line) } catch { return null }
        })
        .filter(item => item !== null)

      if (rawData.length === 0) return await e.reply("数据为空", true)

      // 按群组分组
      const groups = lodash.groupBy(rawData, "group_id")
      
      let qaPairs = [] // 存放对话对
      let rawTexts = [] // 存放纯文本

      // 遍历每个群的消息
      for (const groupId in groups) {
        // 按时间排序
        const msgs = groups[groupId].sort((a, b) => a.time - b.time)
        
        // 构建对话链
        const processedIndices = new Set() // 记录已被处理进QA对的索引

        for (let i = 0; i < msgs.length; i++) {
          if (processedIndices.has(i)) continue

          const currentMsg = msgs[i]
          let nextMsgIndex = -1

          // 寻找后面是否有消息显式回复了当前消息
          const replyMsgIndex = msgs.findIndex((m, idx) => idx > i && m.reply_id === currentMsg.msg_id)
          
          if (replyMsgIndex !== -1) {
            nextMsgIndex = replyMsgIndex
          } 
          // 隐式 N-gram 上下文关联
          else if (i + 1 < msgs.length) {
            const potentialNext = msgs[i + 1]
            // 时间差在窗口内，且不是同一个人发的
            if ((potentialNext.time - currentMsg.time) < TIME_WINDOW && potentialNext.user_id !== currentMsg.user_id) {
               // 使用 N-gram 简单判断是否存在语义承接可能性
               // 这里只是一个简单的启发式规则：如果是问句，或者 n-gram 有重叠(虽然对话不一定重叠)，
               // 或者仅仅是短时间内的接话，我们视为弱关联。
               // 为了提高质量，这里计算简单的 Jaccard 相似度来避免完全无关的话题，
               // 但对于聊天记录，往往不需要高相似度。这里更多通过 n-gram 过滤掉完全乱码或复读机。
               if (!this.isRepeater(currentMsg.content, potentialNext.content)) {
                   nextMsgIndex = i + 1
               }
            }
          }

          if (nextMsgIndex !== -1) {
            // 形成 QA 对
            qaPairs.push({
              instruction: currentMsg.content,
              output: msgs[nextMsgIndex].content,
              meta: { group: groupId, time_gap: msgs[nextMsgIndex].time - currentMsg.time }
            })
            processedIndices.add(i)
            processedIndices.add(nextMsgIndex)
            // 跳过已匹配的下一条，避免重叠
            if (nextMsgIndex === i + 1) i++ 
          } else {
            // 无法配对，归入 raw
            rawTexts.push(currentMsg.content)
          }
        }
      }

      const totalUnits = rawTexts.length + (qaPairs.length * 2)
      const targetQACount = Math.floor(totalUnits * 0.2 / 2)

      let finalQA = []
      let finalRaw = [...rawTexts]

      qaPairs = lodash.shuffle(qaPairs)

      if (qaPairs.length > targetQACount) {
        finalQA = qaPairs.slice(0, targetQACount)
        const toDump = qaPairs.slice(targetQACount)
        toDump.forEach(pair => {
            finalRaw.push(pair.instruction)
            finalRaw.push(pair.output)
        })
      } else {
        finalQA = qaPairs
      }
      
      finalRaw = lodash.shuffle(finalRaw)

      const timeStr = moment().format("YYYYMMDD_HHmmss")
      
      const rawPath = path.join(EXPORT_DIR, `raw_corpus_${timeStr}.txt`)
      fs.writeFileSync(rawPath, finalRaw.join("\n\n"), "utf-8")

      const qaPath = path.join(EXPORT_DIR, `qa_corpus_${timeStr}.jsonl`)
      const qaContent = finalQA.map(p => JSON.stringify({
        input: p.instruction,
        output: p.output
      })).join("\n")
      fs.writeFileSync(qaPath, qaContent, "utf-8")

      const report = [
        `「语料生成完成」`,
        `原始记录条数：${rawData.length}`,
        `Raw语料 (80%目标)：${finalRaw.length} 行`,
        `QA语料 (20%目标)：${finalQA.length} 对`,
        `保存路径：${EXPORT_DIR}`
      ].join("\n")

      await e.reply(report, true)
      if (!e.isGroup) {
          if (finalRaw.length > 0) await e.reply(segment.file(rawPath))
          if (finalQA.length > 0) await e.reply(segment.file(qaPath))
      }

    } catch (err) {
      logger.error(`[${PLUGIN_NAME}] 生成语料出错:`, err)
      await e.reply(`处理失败: ${err.message}`, true)
    }
  }

  /**
   * PII Masking: 敏感信息脱敏
   */
  piiMasking(text) {
    if (!text) return ""
    return text
      .replace(REGEX_PHONE, "[PHONE_REMOVED]")
      .replace(REGEX_EMAIL, "[EMAIL_REMOVED]")
      .replace(REGEX_ID_CARD, "[ID_REMOVED]")
  }

  /**
   * 打死复读机
   * 利用 N-gram (Bigram) 计算 Jaccard 相似度
   */
  isRepeater(str1, str2) {
    if (str1 === str2) return true // 完全复读

    const grams1 = this.getNGrams(str1, N_GRAM_SIZE)
    const grams2 = this.getNGrams(str2, N_GRAM_SIZE)
    
    if (grams1.size === 0 || grams2.size === 0) return false

    const intersection = new Set([...grams1].filter(x => grams2.has(x)))
    const union = new Set([...grams1, ...grams2])
    
    const jaccardIndex = intersection.size / union.size
    
    // 如果相似度过高（例如 > 0.8），可能是在复读或者稍微改字的复读，不适合做QA
    return jaccardIndex > 0.8
  }

  /**
   * 生成 N-gram Set
   */
  getNGrams(text, n) {
    const grams = new Set()
    if (!text || text.length < n) return grams
    
    for (let i = 0; i <= text.length - n; i++) {
      grams.add(text.substring(i, i + n))
    }
    return grams
  }
}