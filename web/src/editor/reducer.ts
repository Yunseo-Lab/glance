import type {Board,Source,Widget} from '../types';
export type EditorState={saved?:Board;draft?:Board;editing:boolean};
export type EditorCommand=
 |{type:'load';board:Board}|{type:'clear'}|{type:'begin'}|{type:'cancel'}
 |{type:'replace';board:Board}|{type:'title';title:string}|{type:'theme';theme:Board['theme']}
 |{type:'upsert';widget:Widget;source?:Source}|{type:'remove';id:string}
 |{type:'move';id:string;to:number}|{type:'duplicate';id:string;newID:string};
export const initialEditor:EditorState={editing:false};
export function pruneBoard(board:Board):Board{const used=new Set(board.widgets.map(widget=>widget.sourceId));return {...board,sources:board.sources.filter(source=>used.has(source.id))}}
export function editorReducer(state:EditorState,command:EditorCommand):EditorState{
 if(command.type==='clear')return initialEditor;
 if(command.type==='load')return {saved:structuredClone(command.board),draft:structuredClone(command.board),editing:false};
 if(command.type==='begin')return state.draft?{...state,editing:true}:state;
 if(command.type==='cancel')return {...state,draft:state.saved?structuredClone(state.saved):undefined,editing:false};
 const board=state.draft;if(!board)return state;
 let next:Board=board;
 switch(command.type){
  case 'replace':if(command.board.id!==board.id||command.board.revision!==board.revision)return state;next=structuredClone(command.board);break;
  case 'title':next={...board,title:command.title};break;
  case 'theme':next={...board,theme:command.theme};break;
  case 'upsert':{
   const index=board.widgets.findIndex(widget=>widget.id===command.widget.id);if(index<0&&board.widgets.length>=32)return state;
   const widgets=board.widgets.map(widget=>widget.id===command.widget.id?structuredClone(command.widget):widget);if(index<0)widgets.push(structuredClone(command.widget));
   const sources=command.source?[...board.sources.filter(source=>source.id!==command.source!.id),structuredClone(command.source)]:board.sources;
   if(command.widget.sourceId&&!sources.some(source=>source.id===command.widget.sourceId))return state;
   next=pruneBoard({...board,widgets,sources});break;
  }
  case 'remove':if(!board.widgets.some(widget=>widget.id===command.id))return state;next=pruneBoard({...board,widgets:board.widgets.filter(widget=>widget.id!==command.id)});break;
  case 'move':{
   const from=board.widgets.findIndex(widget=>widget.id===command.id);if(from<0||!Number.isInteger(command.to)||command.to<0||command.to>=board.widgets.length||from===command.to)return state;
   const widgets=[...board.widgets];widgets.splice(command.to,0,widgets.splice(from,1)[0]);next={...board,widgets};break;
  }
  case 'duplicate':{
   const index=board.widgets.findIndex(widget=>widget.id===command.id);if(index<0||board.widgets.length>=32||!command.newID||board.widgets.some(widget=>widget.id===command.newID))return state;
   const copy=structuredClone(board.widgets[index]);copy.id=command.newID;let title='';for(const char of copy.title){if(new TextEncoder().encode(title+char).length>90)break;title+=char}copy.title=title+' 복사';const widgets=[...board.widgets];widgets.splice(index+1,0,copy);next={...board,widgets};break;
  }
 }
 return next===board?state:{...state,draft:next};
}
