import { reactive, readonly } from 'vue'

const DIAGNOSIS_ACTIONS = new Set(['save', 'amend', 'commit', 'abate'])
const ORDER_ACTIONS = new Set(['save', 'commit', 'abate'])
const CHARGE_ACTIONS = new Set(['pay', 'refund'])

export function createSubmitFeedback() {
  const state = reactive({
    status: 'idle',
    action: '',
    message: '',
    code: '',
    registerId: '',
    registerNo: '',
    diagnoseId: '',
    orderId: '',
    orderNo: '',
    chargeId: '',
    squareNo: '',
  })

  function beginAttempt(action) {
    Object.assign(state, {
      status: 'submitting',
      action: action || '',
      message: '',
      code: '',
      registerId: '',
      registerNo: '',
      diagnoseId: '',
      orderId: '',
      orderNo: '',
      chargeId: '',
      squareNo: '',
    })
  }

  function succeed(result, message) {
    const registerId = typeof result?.registerId === 'string' ? result.registerId : ''
    const registerNo = typeof result?.registerNo === 'string' ? result.registerNo : ''
    const diagnoseId = typeof result?.diagnoseId === 'string' ? result.diagnoseId : ''
    const orderId = typeof result?.orderId === 'string' ? result.orderId : ''
    const orderNo = typeof result?.orderNo === 'string' ? result.orderNo : ''
    const chargeId = typeof result?.chargeId === 'string' ? result.chargeId : ''
    const squareNo = typeof result?.squareNo === 'string' ? result.squareNo : ''
    let defaultMessage = '操作成功'
    if (DIAGNOSIS_ACTIONS.has(state.action) && diagnoseId) {
      const statusHint = {
        save: '已暂存',
        amend: '修订已暂存',
        commit: '已提交',
        abate: '已作废',
      }[state.action] || '操作成功'
      defaultMessage = `诊断${statusHint}，诊断号 ${diagnoseId}`
    } else if ((ORDER_ACTIONS.has(state.action) || orderNo || orderId) && (orderNo || orderId) && !CHARGE_ACTIONS.has(state.action) && !chargeId && !squareNo) {
      const statusHint = {
        save: '已暂存',
        commit: '已提交',
        abate: '已作废',
      }[state.action] || '操作成功'
      const identifier = orderNo || orderId
      defaultMessage = `处方${statusHint}，流水号 ${identifier}`
    } else if (CHARGE_ACTIONS.has(state.action) || chargeId || squareNo) {
      const identifier = squareNo || chargeId
      if (state.action === 'refund') {
        defaultMessage = identifier ? `已退费。收费单号 ${identifier}` : '已退费'
      } else {
        defaultMessage = identifier ? `收费成功，可去发药。收费单号 ${identifier}` : '收费成功，可去发药。'
      }
    } else {
      const identifier = registerNo || registerId
      if (identifier) {
        defaultMessage = (state.action === 'register' ? '挂号成功' : '操作成功') + '，流水号 ' + identifier
      }
    }
    Object.assign(state, {
      status: 'success',
      message: typeof message === 'string' && message.trim() ? message.trim() : defaultMessage,
      code: '',
      registerId,
      registerNo,
      diagnoseId,
      orderId,
      orderNo,
      chargeId,
      squareNo,
    })
  }

  function fail(message, code = '') {
    Object.assign(state, {
      status: 'error',
      message: typeof message === 'string' && message.trim() ? message.trim() : '操作失败，请稍后重试',
      code: typeof code === 'string' ? code : '',
      registerId: '',
      registerNo: '',
      diagnoseId: '',
      orderId: '',
      orderNo: '',
      chargeId: '',
      squareNo: '',
    })
  }

  function reset() {
    Object.assign(state, {
      status: 'idle',
      action: '',
      message: '',
      code: '',
      registerId: '',
      registerNo: '',
      diagnoseId: '',
      orderId: '',
      orderNo: '',
      chargeId: '',
      squareNo: '',
    })
  }

  return { state: readonly(state), beginAttempt, succeed, fail, reset }
}
