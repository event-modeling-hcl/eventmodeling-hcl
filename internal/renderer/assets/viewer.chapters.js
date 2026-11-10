/* Chapter overview: one card per chapter, previewing its first and last slice.
   The group of board workflows in no chapter carries a viewer label, styled apart. */
(function(){
  function sliceSummary(slice){
    const node = document.createElement("section");
    node.className = "chapter-slice";
    const heading = document.createElement("h3");
    heading.textContent = slice.title;
    node.append(heading, EMC.sliceTimeline(slice));
    return node;
  }
  window.renderChapters = function(){
    const host = document.getElementById("board-chapters");
    const source = window.EMC && EMC.FULL_MODEL;
    if(!host || !source) return;
    host.replaceChildren();
    host.hidden = false;
    const slices = new Map((source.slices || []).map(slice=>[slice.id,slice]));
    const records = EMC.chapters || [];
    if(!records.length){
      const empty = document.createElement("p");
      empty.className = "chapter-empty-model";
      empty.textContent = "No chapters are available.";
      host.appendChild(empty);
      return;
    }
    const list = document.createElement("div");
    list.className = "chapter-list";
    records.forEach((chapter,index)=>{
      const card = document.createElement("article");
      card.className = "chapter-overview" + (chapter.ui ? " ui-group" : "");
      const sliceIDs = (chapter.slices || []).filter(id=>slices.has(id));
      const header = document.createElement("div");
      header.className = "chapter-overview-header";
      const number = document.createElement("span");
      number.className = "chapter-number mono";
      number.textContent = chapter.ui ? "" : String(index+1).padStart(2,"0");
      const title = document.createElement("h2");
      // One link per card; CSS stretches its hit area over the whole card.
      const link = document.createElement("a");
      link.className = "chapter-link" + (chapter.ui ? " ui-label" : "");
      link.href = EMC.chapterHref(chapter.id) + "/model";
      link.textContent = chapter.title;
      title.appendChild(link);
      const count = document.createElement("span");
      count.className = "chapter-count mono";
      count.textContent = `${sliceIDs.length} ${sliceIDs.length===1?"slice":"slices"}`;
      header.append(number,title,count);
      card.appendChild(header);
      const preview = document.createElement("div");
      const ordered = sliceIDs.map(id=>slices.get(id));
      preview.className = "chapter-preview " + (ordered.length===1 ? "one-slice" : ordered.length===2 ? "two-slices" : "many-slices");
      if(!ordered.length){
        const noSlices = document.createElement("p");
        noSlices.className = "chapter-empty";
        noSlices.textContent = "No slices";
        preview.appendChild(noSlices);
      } else {
        const addSlice = slice => preview.appendChild(sliceSummary(slice));
        addSlice(ordered[0]);
        if(ordered.length > 2){
          const omitted = ordered.length-2;
          const omission = document.createElement("div");
          omission.className = "chapter-omission mono";
          omission.innerHTML = `<span aria-hidden="true">···</span><span>${omitted} more ${omitted===1?"slice":"slices"}</span>`;
          preview.appendChild(omission);
        }
        if(ordered.length > 1) addSlice(ordered[ordered.length-1]);
      }
      card.appendChild(preview);
      list.appendChild(card);
    });
    host.appendChild(list);
  };
})();

/* The index reads the full board, never a chapter-scoped copy. */
(function(){
  let rendered = false;
  const host = document.getElementById("board-index");
  const track = document.getElementById("index-track");
  document.getElementById("index-flows").addEventListener("change", event=>{
    host.classList.toggle("show-flows", event.target.checked);
  });
  window.renderIndex = function(){
    if(rendered) return;
    rendered = true;
    const {FULL_MODEL, chapters, el, esc} = EMC;
    const slices = FULL_MODEL.slices || [];
    const positions = new Map(slices.map((slice,index)=>[slice.id,index+1]));
    const groups = chapters.length ? chapters.map(chapter=>{
      const members = new Set(chapter.slices);
      return {chapter, slices:slices.filter(slice=>members.has(slice.id))};
    }) : [{chapter:null, slices}];
    groups.forEach(({chapter,slices})=>{
      if(!slices.length) return;
      const group = el("section","index-group");
      const heading = el("h3","index-chapter");
      if(chapter){
        const link = el("a","",esc(chapter.title));
        link.href = EMC.chapterHref(chapter.id) + "/model";
        if(!chapter.ui) heading.append("Chapter ");
        heading.append(link,":");
        if(chapter.ui) heading.classList.add("ui-label");
      } else heading.textContent = "Story:";
      const list = el("ol","index-slices");
      group.append(heading,list);
      track.appendChild(group);
      slices.forEach(slice=>{
        const item = el("li","index-slice");
        item.dataset.sliceId = slice.id;
        const button = el("button","index-title");
        button.type = "button";
        button.textContent = slice.title;
        button.addEventListener("click",()=>EMC.openSliceDetail(slice,"slice " + String(positions.get(slice.id)).padStart(2,"0")));
        item.appendChild(button);
        const flow = el("div","index-flow");
        const stages = new Map();
        const preview = EMC.chapterSlices(FULL_MODEL,{slices:[slice.id]})[0];
        preview.elements.filter(element=>["screen","command","event","readmodel","processor"].includes(element.kind))
          .forEach(element=>{
            const stage = element.stage || 0;
            if(!stages.has(stage)) stages.set(stage,[]);
            stages.get(stage).push(element);
          });
        [...stages.entries()].sort((a,b)=>a[0]-b[0]).forEach(([stage,elements])=>{
          const column = el("div","index-stage");
          elements.forEach(element=>{
            const node = el("div",`timeline-node ${element.kind}${element.external ? " external" : ""}`);
            const label = element.kind === "processor" ? "Automation" : EMC.kindLabel(element);
            node.append(el("span","timeline-node-kind",esc(label)),el("span","timeline-node-title",esc(element.title)));
            column.appendChild(node);
          });
          flow.appendChild(column);
        });
        if(!stages.size) flow.textContent = "No flow elements";
        item.appendChild(flow);
        list.appendChild(item);
      });
    });
    if(!track.childElementCount) track.appendChild(el("p","chapter-empty","No slices on the board."));
  };
})();
