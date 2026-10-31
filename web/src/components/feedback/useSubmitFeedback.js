import { reactive, readonly } from 'vue'

const DIAGNOSIS_ACTIONS = new Set(['save', 'amend', 'commit', 'abate'])
const ORDER_ACTIONS = new Set(['save', 'commit', 'abate'])
const CHARGE_ACTIONS = new Set(['pay', 'refund'])
const DISPENSE_ACTIONS = new Set(['dispense', 'return'])

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
    dispenseId: '',
    dispenseNo: '',
  })

  function clearIds() {
    return {
      registerId: '',
      registerNo: '',
      diagnoseId: '',
      orderId: '',
      orderNo: '',
      chargeId: '',
      squareNo: '',
      dispenseId: '',
      dispenseNo: '',
    }
  }

  function beginAttempt(action) {
    Object.assign(state, {
      status: 'submitting',
      action: action || '',
      message: '',
      code: '',
      ...clearIds(),
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
    const dispenseId = typeof result?.dispenseId === 'string' ? result.dispenseId : ''
    const dispenseNo = typeof result?.dispenseNo === 'string' ? result.dispenseNo : ''
    let defaultMessage = '操作成功'
    if (DIAGNOSIS_ACTIONS.has(state.action) && diagnoseId) {
      const statusHint = {
        save: '已暂存',
        amend: '修订已暂存',
        commit: '已提交',
        abate: '已作废',
      }[state.action] || '操作成功'
      defaultMessage = `诊断${statusHint}，诊断号 ${diagnoseId}`
    } else if ((ORDER_ACTIONS.has(state.action) || orderNo || orderId) && (orderNo || orderId) && !CHARGE_ACTIONS.has(state.action) && !chargeId && !squareNo && !DISPENSE_ACTIONS.has(state.action) && !dispenseId && !dispenseNo) {
      const statusHint = {
        save: '已暂存',
        commit: '已提交',
        abate: '已作废',
      }[state.action] || '操作成功'
      const identifier = orderNo || orderId
      defaultMessage = `处方${statusHint}，流水号 ${identifier}`
    } else if (CHARGE_ACTIONS.has(state.action) || ((chargeId || squareNo) && !DISPENSE_ACTIONS.has(state.action) && !dispenseId && !dispenseNo)) {
      const identifier = squareNo || chargeId
      if (state.action === 'refund') {
        defaultMessage = identifier ? `已退费。收费单号 ${identifier}` : '已退费'
      } else {
        defaultMessage = identifier ? `收费成功，可去发药。收费单号 ${identifier}` : '收费成功，可去发药。'
      }
    } else if (DISPENSE_ACTIONS.has(state.action) || dispenseId || dispenseNo) {
      const identifier = dispenseNo || dispenseId
      if (state.action === 'return') {
        defaultMessage = identifier ? `已退药。发药单号 ${identifier}` : '已退药'
      } else {
        defaultMessage = identifier ? `已发药，发药单号 ${identifier}` : '已发药'
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
      dispenseId,
      dispenseNo,
    })
  }

  function fail(message, code = '') {
    Object.assign(state, {
      status: 'error',
      message: typeof message === 'string' && message.trim() ? message.trim() : '操作失败，请稍后重试',
      code: typeof code === 'string' ? code : '',
      ...clearIds(),
    })
  }

  function reset() {
    Object.assign(state, {
      status: 'idle',
      action: '',
      message: '',
      code: '',
      ...clearIds(),
    })
  }

  return { state: readonly(state), beginAttempt, succeed, fail, reset }
}
