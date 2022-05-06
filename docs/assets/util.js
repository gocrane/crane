setTimeout(function () {
  const requestAnimationFrame = window.requestAnimationFrame;
  const requestIdleCallback = window.requestIdleCallback;
  requestIdleCallback(()=> {
    requestAnimationFrame(()=> {
      // Remove # when use markdown annotations
      const mdAnnotations = document.querySelectorAll(".md-annotation")
      for (let i = 0; i < mdAnnotations.length; i++) {
        let tmp = mdAnnotations[i]
        let parentChinldNodes = tmp.parentElement.childNodes
        if (parentChinldNodes[0].data === '#'){
          parentChinldNodes[0].remove()
        }
      }

      // Handle tab label click
      const labels = document.querySelectorAll("div.tabbed-labels > label")
      for (let i = 0; i < labels.length; i++) {
        let tmp = labels[i]
        tmp.onclick = () => {
          for (const label of labels) {
            if (label.textContent === tmp.textContent) {
              label.click()
            }
          }
        }
      }
    })
  })
},1)
