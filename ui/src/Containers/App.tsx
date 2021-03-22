/* eslint-disable @typescript-eslint/no-unused-vars */
/* eslint-disable no-unused-vars */
/* eslint-disable @typescript-eslint/no-explicit-any */
/* eslint-disable @typescript-eslint/no-use-before-define */
// todo : clean types
import React, { useRef, useEffect } from 'react'
import * as d3 from 'd3'
import { depthCount, flareData } from '../lib/utils'

interface IProps {
  data?: any // todo : add type
}

export const MyD3Component: React.FC<IProps> = props => {
  const d3Container = useRef(null)
  useEffect(
    () => {
      if (props.data) {
        const data = flareData(props.data)
        const width = 932
        const height = 932

        const color = d3
          .scaleLinear()
          .domain([0, depthCount(data)])
          .rangeRound([0, 500])
          .interpolate(() => d3.interpolateHcl('hsl(152,80%,80%)', 'hsl(228,30%,40%)'))

        d3.format(',d')

        const pack = (data: any): any =>
          d3
            .pack()
            .size([width, height])
            .radius(d => {
              return 45
            })
            .padding(30)(
            d3
              .hierarchy(data)
              .sum(d => d.value)
              .sort((a, b) => (a.value && b.value ? b.value - a.value : 0)),
          )

        const root = pack(data)
        let focus = root
        let view: any

        const svg = d3
          .select(d3Container.current)
          .attr('viewBox', `-${width / 2} -${height / 2} ${width} ${height}`)
          .style('display', 'block')
          .style('margin', '0 -14px')
          .style('background', color(0))
          .style('cursor', 'pointer')
          .on('click', event => zoom(event, root))

        const node = svg
          .append('g')
          .selectAll('circle')
          .data(root)
          .join('circle')
          .attr('fill', (d: any) => (d.children ? color(d.depth) : 'white'))
          .attr('pointer-events', (d: any) => (!d.children ? 'none' : null))
          .on('mouseover', function() {
            d3.select(this).attr('stroke', '#000')
          })
          .on('mouseout', function() {
            d3.select(this).attr('stroke', null)
          })
          .on('click', (event, d) => focus !== d && (zoom(event, d), event.stopPropagation()))

        const label = svg
          .append('g')
          .style('font', '10px sans-serif')
          .attr('pointer-events', 'none')
          .attr('text-anchor', 'middle')
          .selectAll('text')
          .data(root)
          .join('text')
          .style('fill-opacity', (d: any) => (d.parent === root ? 1 : 0))
          .style('display', (d: any) => (d.parent === root ? 'inline' : 'none'))
          .text((d: any) => d.data.name)

        const zoomTo = (v: any): any => {
          const k = width / v[2]

          view = v

          label.attr('transform', (d: any) => `translate(${(d.x - v[0]) * k},${(d.y - v[1]) * k})`)
          node.attr('transform', (d: any) => `translate(${(d.x - v[0]) * k},${(d.y - v[1]) * k})`)
          node.attr('r', (d: any) => d.r * k)
        }

        const zoom = (event: any, d: any): any => {
          const focus0 = focus

          focus = d

          const transition = svg
            .transition()
            .duration(event.altKey ? 7500 : 750)
            .tween('zoom', (d: any) => {
              const i = d3.interpolateZoom(view, [focus.x, focus.y, focus.r * 2])
              return (t: any) => zoomTo(i(t))
            })

          label
            .filter(function(d: any) {
              return d.parent === focus || (this as SVGTextElement).style.display === 'inline'
            })
            .transition()
            .style('fill-opacity', (d: any) => {
              return d.parent === focus ? 1 : 0
            })
            .on('start', function(d: any) {
              if (d.parent === focus) (this as SVGTextElement).style.display = 'inline'
            })
            .on('end', function(d: any) {
              if (d.parent !== focus) (this as SVGTextElement).style.display = 'none'
            })
        }
        zoomTo([root.x, root.y, root.r * 2])
      }
    },

    // eslint-disable-next-line react-hooks/exhaustive-deps
    [props.data],
  )

  return <svg height="1600" width="1600" className="d3-component" ref={d3Container}></svg>
}
const App: React.FC = () => {
  const data = [
    // todo : this should be in a state or something
    {
      name: 'service-a',
      uptime: 12312321,
      instances: ['host1:port1', 'host2:port2'],
    },
    {
      name: 'service-b',
      uptime: 32319321,
      instances: ['host1:port1', 'host2:port2'],
    },
    {
      name: 'service-b',
      uptime: 12312321,
      instances: ['host1:port1', 'host2:port2'],
    },
  ]
  return (
    <div id="container">
      <MyD3Component data={data} />
    </div>
  )
}

export default App
