export function normalizeImageModelValue(modelValue, multiple, maxCount) {
  if (multiple || maxCount > 1) {
    return Array.isArray(modelValue) ? modelValue : []
  }
  if (Array.isArray(modelValue)) {
    return modelValue.filter(Boolean).slice(0, 1)
  }
  return modelValue ? [modelValue] : []
}

export function isImageFileTooLarge(file, maxSizeMb) {
  return file.size > maxSizeMb * 1024 * 1024
}
