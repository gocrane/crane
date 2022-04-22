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
    })
  })
},1)
